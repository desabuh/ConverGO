package history

import (
	"fmt"
	"testing"

	"github.com/desabuh/convergo/cvrdt"
	"github.com/desabuh/convergo/utils"
	"github.com/stretchr/testify/assert"
)

// TestDAGBasicConstruction tests basic DAG construction from a simple operation sequence
func TestDAGBasicConstruction(t *testing.T) {
	const SITE_ID = "1"

	// Create a site and perform sequential operations
	crdt := cvrdt.NewWootCvrdtWithView(SITE_ID)

	// Insert "A"
	err := crdt.UpdateState(cvrdt.GetNewStateFromOp(
		cvrdt.CreateNewLocalOp(cvrdt.Insertion, 0, "A", SITE_ID),
	))
	assert.Nil(t, err)

	// Insert "B" (depends on A - same site sequential operations)
	err = crdt.UpdateState(cvrdt.GetNewStateFromOp(
		cvrdt.CreateNewLocalOp(cvrdt.Insertion, 1, "B", SITE_ID),
	))
	assert.Nil(t, err)

	// Build DAG
	state := crdt.GetState()
	dag := BuildDAG(state)

	// Verify DAG structure
	assert.Equal(t, 2, len(dag.Nodes), "Should have 2 operation nodes")

	// Verify operations are in the DAG
	op1Id := "1_1"
	op2Id := "1_2"
	assert.NotNil(t, dag.Nodes[op1Id], "Operation 1 should be in DAG")
	assert.NotNil(t, dag.Nodes[op2Id], "Operation 2 should be in DAG")

	//Verfy only op1Id is root
	assert.Len(t, dag.Roots, 1)
	if _, found := dag.Roots[op1Id]; !found {
		t.Fatalf("operation %s should be a root in dag", op1Id)
	}

	// Verify causality: op1 → op2 (op2 depends on op1)
	assert.True(t, dag.HasPath(op1Id, op2Id), "op1 should happen-before op2")
	assert.False(t, dag.HasPath(op2Id, op1Id), "op2 should not happen-before op1")
	assert.False(t, dag.IsConcurrent(op1Id, op2Id), "op1 and op2 should not be concurrent")

	t.Logf("DAG structure:\n%s", dag.String())

}

// TestDAGConcurrentOperations tests DAG with concurrent operations from different sites
func TestDAGConcurrentOperations(t *testing.T) {
	const SITE_ID_1 = "1"
	const SITE_ID_2 = "2"

	// Create two sites
	site1 := cvrdt.NewWootCvrdtWithView(SITE_ID_1)
	site2 := cvrdt.NewWootCvrdtWithView(SITE_ID_2)

	// Site 1: Insert "A"
	err := site1.UpdateState(cvrdt.GetNewStateFromOp(
		cvrdt.CreateNewLocalOp(cvrdt.Insertion, 0, "A", SITE_ID_1),
	))
	assert.Nil(t, err)

	// Site 2: Insert "B" (concurrent with A - no causal dependency)
	err = site2.UpdateState(cvrdt.GetNewStateFromOp(
		cvrdt.CreateNewLocalOp(cvrdt.Insertion, 0, "B", SITE_ID_2),
	))
	assert.Nil(t, err)

	// Merge states
	err = site1.UpdateState(site2.GetState())
	assert.Nil(t, err)

	// Build DAG from site1
	dag := BuildDAG(site1.GetState())

	// Verify both operations are in DAG
	op1Id := "1_1"
	op2Id := "2_1"
	assert.NotNil(t, dag.Nodes[op1Id])
	assert.NotNil(t, dag.Nodes[op2Id])

	//in this case both op1Id and op2Id should be root
	assert.Len(t, dag.Roots, 2)
	if _, found := dag.Roots[op1Id]; !found {
		t.Fatalf("operation %s should be a root in dag", op1Id)
	}
	if _, found := dag.Roots[op2Id]; !found {
		t.Fatalf("operation %s should be a root in dag", op2Id)
	}

	// Verify they are concurrent (neither happened-before the other)
	assert.False(t, dag.HasPath(op1Id, op2Id), "A and B should be concurrent")
	assert.False(t, dag.HasPath(op2Id, op1Id), "A and B should be concurrent")
	assert.True(t, dag.IsConcurrent(op1Id, op2Id), "A and B should be concurrent")

	t.Logf("DAG with concurrent operations:\n%s", dag.String())
}

// TestDAGConvergence tests that two sites build identical DAGs after merging
func TestDAGConvergence(t *testing.T) {
	const SITE_ID_1 = "1"
	const SITE_ID_2 = "2"

	// Create two sites
	site1 := cvrdt.NewWootCvrdtWithView(SITE_ID_1)
	site2 := cvrdt.NewWootCvrdtWithView(SITE_ID_2)

	// Site 1 operations
	site1.UpdateState(cvrdt.GetNewStateFromOp(
		cvrdt.CreateNewLocalOp(cvrdt.Insertion, 0, "H", SITE_ID_1),
	))
	site1.UpdateState(cvrdt.GetNewStateFromOp(
		cvrdt.CreateNewLocalOp(cvrdt.Insertion, 1, "i", SITE_ID_1),
	))

	// Site 2 operations
	site2.UpdateState(cvrdt.GetNewStateFromOp(
		cvrdt.CreateNewLocalOp(cvrdt.Insertion, 0, "!", SITE_ID_2),
	))

	// Merge: Site 1 receives Site 2's state
	err := site1.UpdateState(site2.GetState())
	assert.Nil(t, err)

	// Merge: Site 2 receives Site 1's state
	err = site2.UpdateState(site1.GetState())
	assert.Nil(t, err)

	// Build DAGs from both sites
	dag1 := BuildDAG(site1.GetState())
	dag2 := BuildDAG(site2.GetState())

	// CONVERGENCE: Both DAGs should be identical
	assert.True(t, dag1.Equals(dag2), "DAGs from both sites should be identical after convergence")

	// Verify structure
	assert.Equal(t, len(dag1.Nodes), len(dag2.Nodes), "Same number of nodes")

	t.Logf("Site 1 DAG:\n%s", dag1.String())
	t.Logf("Site 2 DAG:\n%s", dag2.String())
}

// TestDAGVisualization tests the ASCII tree visualization
func TestDAGVisualization(t *testing.T) {
	const SITE_ID_1 = "1"
	const SITE_ID_2 = "2"
	const SITE_ID_3 = "3"

	// Create three sites
	site1 := cvrdt.NewWootCvrdtWithView(SITE_ID_1)
	site2 := cvrdt.NewWootCvrdtWithView(SITE_ID_2)
	site3 := cvrdt.NewWootCvrdtWithView(SITE_ID_3)

	// Site 1: Insert "Hi"
	site1.UpdateState(cvrdt.GetNewStateFromOp(
		cvrdt.CreateNewLocalOp(cvrdt.Insertion, 0, "H", SITE_ID_1),
	))
	site1.UpdateState(cvrdt.GetNewStateFromOp(
		cvrdt.CreateNewLocalOp(cvrdt.Insertion, 1, "i", SITE_ID_1),
	))

	// Site 2: Insert "W" (concurrent with Site 1)
	site2.UpdateState(cvrdt.GetNewStateFromOp(
		cvrdt.CreateNewLocalOp(cvrdt.Insertion, 0, "W", SITE_ID_2),
	))

	// Site 3: Insert "!" (concurrent)
	site3.UpdateState(cvrdt.GetNewStateFromOp(
		cvrdt.CreateNewLocalOp(cvrdt.Insertion, 0, "!", SITE_ID_3),
	))

	// Merge all sites
	site1.UpdateState(site2.GetState())
	site1.UpdateState(site3.GetState())

	// Site 1 adds another operation (depends on previous)
	site1.UpdateState(cvrdt.GetNewStateFromOp(
		cvrdt.CreateNewLocalOp(cvrdt.Insertion, 1, " ", SITE_ID_1),
	))

	// Build DAG
	dag := BuildDAG(site1.GetState())

	// Test ASCII Tree visualization with depth limit of 10
	t.Log("\n=== ASCII Tree (max depth: 10) ===")
	treeOutput := dag.ToASCIITree(10)
	t.Log(treeOutput)
	assert.NotEmpty(t, treeOutput)
	assert.Contains(t, treeOutput, "DAG:")

	// Test ASCII Tree without depth limit
	t.Log("\n=== ASCII Tree (no limit) ===")
	treeOutputUnlimited := dag.ToASCIITree(0)
	t.Log(treeOutputUnlimited)
	assert.NotEmpty(t, treeOutputUnlimited)

	for _, n := range dag.Nodes {
		fmt.Printf("n.Operation.OpId().Clock: %v\n", n.Operation.OpId().Clock)
	}
}

func TestDAGGraph(t *testing.T) {
	site1 := cvrdt.NewWootCvrdtWithView("1")
	site2 := cvrdt.NewWootCvrdtWithView("2")

	site1.UpdateState(cvrdt.GetNewStateFromOp(
		cvrdt.CreateNewLocalOp(cvrdt.Insertion, 0, "Hi", "1"),
	))

	site2.UpdateState(site1.GetState())

	site2.UpdateState(cvrdt.GetNewStateFromOp(
		cvrdt.CreateNewLocalOp(cvrdt.Insertion, 0, "User", "2"),
	))

	fmt.Printf("BuildDAG(site2.GetState()).ToASCIITree(5): %v\n", BuildDAG(site2.GetState()).ToASCIITree(5))

	for _, op := range site2.GetState() {
		fmt.Printf("op.Char(): %v\n", op.Char())
		fmt.Printf("op.OpId().SiteId: %v\n", op.OpId().SiteId)
	}

	fmt.Printf("utils.First[string](site2.Snapshot()): %v\n", utils.First[string](site2.Snapshot()))

}

func TestDAGG2raph(t *testing.T) {
	site1 := cvrdt.NewWootCvrdtWithView("1")

	site1.UpdateState(cvrdt.GetNewStateFromOp(
		cvrdt.CreateNewLocalOp(cvrdt.Insertion, 0, "Hi", "1"),
	))

	site1.UpdateState(cvrdt.GetNewStateFromOp(
		cvrdt.CreateNewLocalOp(cvrdt.Insertion, 0, "User", "1"),
	))

	fmt.Printf("BuildDAG(site1.GetState()).ToASCIITree(5): %v\n", BuildDAG(site1.GetState()).ToASCIITree(5))

	for _, op := range site1.GetState() {
		fmt.Printf("op.Char(): %v\n", op.Char())
		fmt.Printf("op.OpId().SiteId: %v\n", op.OpId().SiteId)
	}

	fmt.Printf("utils.First[string](site2.Snapshot()): %v\n", utils.First[string](site1.Snapshot()))

}

func TestDAGG3raph(t *testing.T) {
	site1 := cvrdt.NewWootCvrdtWithView("1")
	site2 := cvrdt.NewWootCvrdtWithView("2")

	site1.UpdateState(cvrdt.GetNewStateFromOp(
		cvrdt.CreateNewLocalOp(cvrdt.Insertion, 0, "Hi", "1"),
	))

	site2.UpdateState(cvrdt.GetNewStateFromOp(
		cvrdt.CreateNewLocalOp(cvrdt.Insertion, 0, "User", "2"),
	))

	site1.UpdateState(site2.GetState())

	fmt.Printf("BuildDAG(site1.GetState()).ToASCIITree(5): %v\n", BuildDAG(site1.GetState()).ToASCIITree(5))

	for _, op := range site1.GetState() {
		fmt.Printf("op.Char(): %v\n", op.Char())
		fmt.Printf("op.OpId().SiteId: %v\n", op.OpId().SiteId)
	}

	fmt.Printf("utils.First[string](site2.Snapshot()): %v\n", utils.First[string](site1.Snapshot()))

}

func TestDAGConflict(t *testing.T) {
	site1 := cvrdt.NewWootCvrdtWithView("1")
	site2 := cvrdt.NewWootCvrdtWithView("2")

	site1.UpdateState(cvrdt.GetNewStateFromOp(
		cvrdt.CreateNewLocalOp(cvrdt.Insertion, 0, "Hi", "1"),
	))

	site1.UpdateState(cvrdt.GetNewStateFromOp(
		cvrdt.CreateNewLocalOp(cvrdt.Insertion, 0, "How", "1"),
	))

	site2.UpdateState(cvrdt.GetNewStateFromOp(
		cvrdt.CreateNewLocalOp(cvrdt.Insertion, 0, "User", "2"),
	))

	site2.UpdateState(site1.GetState())

	site2.UpdateState(cvrdt.GetNewStateFromOp(
		cvrdt.CreateNewLocalOp(cvrdt.Insertion, 0, "Dy", "2"),
	))

	fmt.Printf("BuildDAG(site2.GetState()).String(): %v\n", BuildDAG(site2.GetState()).String())

	fmt.Printf("BuildDAG(site2.GetState()).String(): %v\n", BuildDAG(site2.GetState()).ToASCIITree(5))
}

// TestDAGVisualizationLargeGraph tests visualization with a larger graph
func TestDAGVisualizationLargeGraph(t *testing.T) {
	const SITE_ID = "1"

	site := cvrdt.NewWootCvrdtWithView(SITE_ID)

	// Insert multiple characters to create a deeper tree
	text := "ABCDEFGHIJ"
	for i, char := range text {
		site.UpdateState(cvrdt.GetNewStateFromOp(
			cvrdt.CreateNewLocalOp(cvrdt.Insertion, i, string(char), SITE_ID),
		))
	}

	// Build DAG
	dag := BuildDAG(site.GetState())

	// Test with depth limit
	t.Log("\n=== Large Graph with Depth Limit (5) ===")
	treeOutput := dag.ToASCIITree(5)
	t.Log(treeOutput)
	assert.Contains(t, treeOutput, "more layers not shown")

	// Test without limit
	t.Log("\n=== Large Graph - Full Tree ===")
	treeOutputFull := dag.ToASCIITree(0)
	t.Log(treeOutputFull)

	// Verify structure
	assert.Equal(t, 10, len(dag.Nodes), "Should have 10 operation nodes")
}
