package history

import (
	"testing"

	"github.com/desabuh/convergo/cvrdt"
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

	//the resulting dag will have two sequential operation
	expectedLin := "DAG with 2 nodes and 1 edges\n\n" +
		"Nodes:\n" +
		"  1_1: 0 parents, 1 children\n" +
		"  1_2: 1 parents, 0 children\n\n" +
		"Edges:\n" +
		"  1_1 → 1_2\n"

	assert.Equal(t, expectedLin, dag.ToASCIILinearization())

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

	//here the resulting DAG is a fragmented one with indipendetely concurrently created operation
	expectedLin := "DAG with 2 nodes and 0 edges\n\n" +
		"Nodes:\n" +
		"  1_1: 0 parents, 0 children\n" +
		"  2_1: 0 parents, 0 children\n\n" +
		"Edges:\n"

	assert.Equal(t, expectedLin, dag.ToASCIILinearization())
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

	//site 1 insert first "H" (id 1_1) and then "i" (id 1_2)
	assert.Equal(t, "H", dag1.Nodes["1_1"].Operation.Char())
	assert.Equal(t, "i", dag1.Nodes["1_2"].Operation.Char())
	//site 2 insert "!" as first operation (id 2_1)
	assert.Equal(t, "!", dag1.Nodes["2_1"].Operation.Char())

	//here site one creates two sequential operation which are concurrent with site two operation
	expectedCommonLin := "DAG with 3 nodes and 1 edges\n\n" +
		"Nodes:\n" +
		"  1_1: 0 parents, 1 children\n" +
		"  1_2: 1 parents, 0 children\n" +
		"  2_1: 0 parents, 0 children\n\n" +
		"Edges:\n" +
		"  1_1 → 1_2\n"

	//we asserted both DAG are equals so checking dag1 is enough
	assert.Equal(t, expectedCommonLin, dag1.ToASCIILinearization())
}

// TestDAGVisualization tests the ASCII tree visualization
func TestPartialDAGVisualization(t *testing.T) {
	const SITE_ID_1 = "1"
	const SITE_ID_2 = "2"
	const SITE_ID_3 = "3"

	// Create three sites
	site1 := cvrdt.NewWootCvrdtWithView(SITE_ID_1)
	site2 := cvrdt.NewWootCvrdtWithView(SITE_ID_2)
	site3 := cvrdt.NewWootCvrdtWithView(SITE_ID_3)

	// Site 1: Insert "Hi" ("H" + "i")
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

	// Test ASCII Tree visualization with depth limit of 2: in thi scenario having the dag 3 topological layers, the last
	//one will be truncated
	treeOutput := dag.ToASCIIDAG(2)

	assert.NotEmpty(t, treeOutput)

	//The first 3 sites operation will be concurrent, second site 1 operation 1_2 will depend on first 1_1 and
	//the successive topological layer will not be showed cause the depth limit was set to 2

	expectedDAGRes :=
		"\n┌─ DAG: 5 operations, 4 causal edges\n" +
			"│\n" +
			"├─ 1_1 [Insertion] 'H' → 1_2\n" +
			"├─ 2_1 [Insertion] 'W' → 1_3\n" +
			"├─ 3_1 [Insertion] '!' → 1_3\n" +
			"│\n" +
			"│  ├─ 1_2 [Insertion] 'i' → 1_3\n" +
			"│  │  ...\n" +
			"│  │  (1 more layers not shown)\n"

	assert.Equal(t, expectedDAGRes, treeOutput)

	// Test ASCII Tree without depth limit (till 10 layers in the example)
	treeOutputUnlimited := dag.ToASCIIDAG(10)
	assert.NotEmpty(t, treeOutputUnlimited)

	//in the extended version, the last site 1 operation happened after merging 2_1 and 3_1 so it has 3 parents
	expectedDAGRes =
		"\n┌─ DAG: 5 operations, 4 causal edges\n" +
			"│\n" +
			"├─ 1_1 [Insertion] 'H' → 1_2\n" +
			"├─ 2_1 [Insertion] 'W' → 1_3\n" +
			"├─ 3_1 [Insertion] '!' → 1_3\n" +
			"│\n" +
			"│  ├─ 1_2 [Insertion] 'i' → 1_3\n" +
			"│  │\n" +
			"│  │  └─ 1_3 [Insertion] ' ' [3 parents (1_2, 2_1, 3_1)]\n"

	assert.Equal(t, expectedDAGRes, treeOutputUnlimited)
}

func TestDAGConflict(t *testing.T) {
	site1 := cvrdt.NewWootCvrdtWithView("1")
	site2 := cvrdt.NewWootCvrdtWithView("2")

	//site 1 insert "Hi"
	site1.UpdateState(cvrdt.GetNewStateFromOp(
		cvrdt.CreateNewLocalOp(cvrdt.Insertion, 0, "Hi", "1"),
	))

	//site 1 deletes in position 0
	site1.UpdateState(cvrdt.GetNewStateFromOp(
		cvrdt.CreateNewLocalOp(cvrdt.Deletion, 0, "", "1"),
	))

	//site 2 insert "User"
	site2.UpdateState(cvrdt.GetNewStateFromOp(
		cvrdt.CreateNewLocalOp(cvrdt.Insertion, 0, "User", "2"),
	))

	//site 2 merge 1
	site2.UpdateState(site1.GetState())

	//site 2 insert "Dy", DAG is reconnected (this operation has two parents)
	site2.UpdateState(cvrdt.GetNewStateFromOp(
		cvrdt.CreateNewLocalOp(cvrdt.Insertion, 0, "Dy", "2"),
	))

	//site1 is not updated with site2 so their DAG differs

	dag1 := BuildDAG(site1.GetState())
	dag2 := BuildDAG(site2.GetState())

	assert.NotEqual(t, len(dag1.Nodes), len(dag2.Nodes))
	//site 1 will have only its two operations but site 2 will have all 4
	assert.Len(t, dag1.Nodes, 2)
	assert.Len(t, dag2.Nodes, 4)

	asciiDag1 := dag1.ToASCIIDAG(5)
	asciiDag2 := dag2.ToASCIIDAG(5)

	assert.NotEqual(t, asciiDag1, asciiDag2)

	//site 1 state should have only its two operations
	expectedDag1Res := "\n┌─ DAG: 2 operations, 1 causal edges\n" +
		"│\n" +
		"├─ 1_1 [Insertion] 'Hi' → 1_2\n" +
		"│\n" +
		"│  └─ 1_2 [Deletion] 'Hi'\n"

	assert.Equal(t, expectedDag1Res, asciiDag1)

	//site2 will have its own additional operations
	expectedDag2Res := "\n┌─ DAG: 4 operations, 3 causal edges\n" +
		"│\n" +
		"├─ 1_1 [Insertion] 'Hi' → 1_2\n" +
		"├─ 2_1 [Insertion] 'User' → 2_2\n" +
		"│\n" +
		"│  ├─ 1_2 [Deletion] 'Hi' → 2_2\n" +
		"│  │\n" +
		"│  │  └─ 2_2 [Insertion] 'Dy' [2 parents (1_2, 2_1)]\n"

	assert.Equal(t, expectedDag2Res, asciiDag2)

	//updating site1 will solve the problem and achieve the same rappresentation
	site1.UpdateState(site2.GetState())

	//now their DAGs will match
	assert.Equal(t, BuildDAG(site1.GetState()).ToASCIIDAG(5), asciiDag2)

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

	//only 5 topologial layers will be showed
	treeOutput := dag.ToASCIIDAG(5)

	expectedDagRes := "\n┌─ DAG: 10 operations, 9 causal edges\n" +
		"│\n" +
		"├─ 1_1 [Insertion] 'A' → 1_2\n" +
		"│\n" +
		"│  ├─ 1_2 [Insertion] 'B' → 1_3\n" +
		"│  │\n" +
		"│  │  ├─ 1_3 [Insertion] 'C' → 1_4\n" +
		"│  │  │\n" +
		"│  │  │  ├─ 1_4 [Insertion] 'D' → 1_5\n" +
		"│  │  │  │\n" +
		"│  │  │  │  ├─ 1_5 [Insertion] 'E' → 1_6\n" +
		"│  │  │  │  │  ...\n" +
		"│  │  │  │  │  (5 more layers not shown)\n"

	assert.Equal(t, expectedDagRes, treeOutput)

}
