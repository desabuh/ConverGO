package history

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/desabuh/convergo/cvrdt"
	"github.com/desabuh/convergo/utils"
)

type OpVisualizationMode int

const (
	DagVisual OpVisualizationMode = iota
	LinearVisual
	ContentVisual
)

// OperationDAG represents a Directed Acyclic Graph of CRDT operations
// Edges represent causal dependencies (happened-before relationships)
type OperationDAG struct {
	Roots map[string]struct{} //contains ids of Nodes without parents (root of the tree)
	Nodes map[string]*DAGNode // a DAGnode is identified by its own id
	Edges map[string][]string // an edge is identified by outgoing DAGnode id and all the possible incoing DAGnode id (concurrent events are possible)
}

// DAGNode represents a single operation in the causal graph
type DAGNode struct {
	Operation cvrdt.CRDTOperation
	Parents   []cvrdt.LogicalId // Operations this depends on (causal predecessors)
	Children  []cvrdt.LogicalId // Operations that depend on this (causal successors)
}

// BuildDAG constructs the causal DAG from a CRDT state
// The DAG structure is deterministic and eventually consistent across all sites
// (CvRDTState have no explicit linearization but its internal information can be use to construct a unique DAG)
func BuildDAG(state cvrdt.CvRDTState) *OperationDAG {
	dag := &OperationDAG{
		Roots: make(map[string]struct{}),
		Nodes: make(map[string]*DAGNode),
		Edges: make(map[string][]string),
	}

	dag.initNodes(state)

	// Second pass: build edges based on vector clock causality
	// Edge A → B means: A happened-before B (B causally depends on A)
	dag.initEdges(state)

	return dag
}

func (dag *OperationDAG) initNodes(ops []cvrdt.CRDTOperation) {
	for _, op := range ops {

		node := &DAGNode{
			Operation: op,
			Parents:   []cvrdt.LogicalId{},
			Children:  []cvrdt.LogicalId{},
		}
		dag.Nodes[op.OpId().String()] = node
	}

}

func (dag *OperationDAG) initEdges(ops []cvrdt.CRDTOperation) {
	for _, opB := range ops {

		bId := opB.OpId().String()

		for _, opA := range ops {

			aId := opA.OpId().String()

			if aId == bId {
				continue // no relation between A and B (no ark hap. before rel between them)
			}

			if opA.OpId().HappenedBefore(opB.OpId()) {
				// then A -> B

				isDirect := true

				for _, opC := range ops {

					cId := opC.OpId().String()

					if cId == aId || cId == bId {
						continue
					}

					// Check if A → C → B (transitive path exists)
					if opA.OpId().HappenedBefore(opC.OpId()) &&
						opC.OpId().HappenedBefore(opB.OpId()) {
						isDirect = false
						break
					}
				}

				if isDirect {
					dag.addEdge(aId, bId)
					dag.Nodes[bId].Parents = append(dag.Nodes[bId].Parents, opA.OpId())
					dag.Nodes[aId].Children = append(dag.Nodes[aId].Children, opB.OpId())
				}

			}

		}

		//if no parent is found after checking all nodes then current node is a root
		if len(dag.Nodes[bId].Parents) == 0 {
			dag.Roots[bId] = struct{}{}
		}

	}
}

// addEdge adds a directed edge from -> to (avoiding duplicates)
func (dag *OperationDAG) addEdge(from, to string) {
	// Check if edge already exists
	for _, existing := range dag.Edges[from] {
		if existing == to {
			return
		}
	}
	dag.Edges[from] = append(dag.Edges[from], to)
}

// HasPath checks if a path between op1 to op2, this is the same as assuming an happened before relation between them exists
// Returns true if there's a path from op1 to op2 in the DAG
func (dag *OperationDAG) HasPath(op1Id, op2Id string) bool {
	node1 := dag.Nodes[op1Id]
	node2 := dag.Nodes[op2Id]

	return node1.Operation.OpId().HappenedBefore(node2.Operation.OpId())
}

// IsConcurrent checks if two operations are causally independent
func (dag *OperationDAG) IsConcurrent(op1Id, op2Id string) bool {
	if op1Id == op2Id {
		return false
	}

	//if no path exists in the OperationDAG then the two Node refers to concurrent operations
	return !dag.HasPath(op1Id, op2Id) && !dag.HasPath(op2Id, op1Id)
}

// return a slice of slice of logicalId where each slice i is the list all operations that depends on layer i-1
func (dag *OperationDAG) GetTopologicalLayers() [][]cvrdt.LogicalId {
	//if nodes are empty return
	if len(dag.Nodes) == 0 {
		return nil
	}

	//indegree: for each nodes is the number of incoming edges
	indegree := make(map[string]int, len(dag.Nodes))
	//starting roots nodes
	current := make([]string, 0, len(dag.Roots))

	// for each nodes their indegree are their direct parents
	for id, node := range dag.Nodes {
		indegree[id] = len(node.Parents)
	}

	//init current as root nodes
	for id := range dag.Roots {
		current = append(current, id)
	}
	//sort starting nodes (first site id and then op id)
	slices.Sort(current)

	var layers [][]cvrdt.LogicalId
	//this var is only neeed to validate incosistent graphs by counting processed nodes(if the graph is a correct DAG no problem will arise)
	processed := 0

	for len(current) > 0 {
		next := make([]string, 0)
		layer := make([]cvrdt.LogicalId, 0, len(current))

		for _, id := range current {
			//init the layer with all the ids of current nodes
			layer = append(layer, dag.Nodes[id].Operation.OpId())
			processed++

			//now find all children of the current operation node
			children := slices.Clone(dag.Edges[id])
			slices.Sort(children) // deterministic unlock order

			//for each children decrease their indegree (the current parent was already processed)
			//if their indegree is 0 it means all its parents are processed so it can be included in the next layer
			for _, child := range children {
				indegree[child]--
				if indegree[child] == 0 {
					next = append(next, child)
				}
			}
		}

		layers = append(layers, layer)
		slices.Sort(next)
		current = next
	}

	if processed != len(dag.Nodes) {
		panic("history: DAG contains a cycle or inconsistent indegrees")
	}

	return layers
}

// Equals compares two DAGs for structural equality
// Used to verify convergence: all sites should have the same DAG
func (dag *OperationDAG) Equals(other *OperationDAG) bool {
	// Check same number of nodes
	if len(dag.Nodes) != len(other.Nodes) {
		return false
	}

	// Check all nodes exist in both
	for opId := range dag.Nodes {
		if _, exists := other.Nodes[opId]; !exists {
			return false
		}
	}

	// Check all edges match
	for from, toList := range dag.Edges {
		otherToList, exists := other.Edges[from]
		if !exists {
			return false
		}

		a, b := slices.Clone(toList), slices.Clone(otherToList)
		slices.Sort(a)
		slices.Sort(b)
		same := slices.Equal(a, b)

		if !same {
			return false
		}
	}

	return true
}

// String provides a human-readable representation of the DAG
func (dag *OperationDAG) String() string {
	var result string
	result += fmt.Sprintf("DAG with %d nodes and %d edges\n", len(dag.Nodes), dag.countEdges())
	result += "\nNodes:\n"

	nodeIds := make([]string, 0, len(dag.Nodes))
	for id := range dag.Nodes {
		nodeIds = append(nodeIds, id)
	}
	sort.Strings(nodeIds) //ids are ordered in alphanumerecal order (first side id and then their own logical id)

	for _, id := range nodeIds {
		node := dag.Nodes[id]
		result += fmt.Sprintf("  %s: %d parents, %d children\n",
			id, len(node.Parents), len(node.Children))
	}

	result += "\nEdges:\n"
	for from, toList := range dag.Edges {
		for _, to := range toList {
			result += fmt.Sprintf("  %s → %s\n", from, to)
		}
	}

	return result
}

func (dag *OperationDAG) countEdges() int {
	count := 0
	for _, toList := range dag.Edges {
		count += len(toList)
	}
	return count
}

func (dag *OperationDAG) ToASCIITree(maxDepth int) string {
	if len(dag.Nodes) == 0 {
		return "Empty DAG\n"
	}

	layers := dag.GetTopologicalLayers()
	if len(layers) == 0 {
		return "Empty DAG\n"
	}

	display := layers
	truncated := false
	if maxDepth > 0 && len(layers) > maxDepth {
		display = layers[:maxDepth]
		truncated = true
	}

	var b strings.Builder
	fmt.Fprintf(&b, "\n┌─ DAG: %d operations, %d causal edges\n", len(dag.Nodes), dag.countEdges())
	b.WriteString("│\n")

	for layerIdx, layer := range display {
		prefix := strings.Repeat("│  ", layerIdx)
		isLastLayer := layerIdx == len(display)-1

		for opIdx, opID := range layer {
			isLastInLayer := opIdx == len(layer)-1
			branch := "├─"
			if isLastLayer && isLastInLayer && !truncated {
				branch = "└─"
			}

			fmt.Fprintf(&b, "%s%s %s\n", prefix, branch, dag.formatASCIINode(opID.String()))
		}

		if !isLastLayer {
			fmt.Fprintf(&b, "%s│\n", prefix)
		}
	}

	if truncated {
		prefix := strings.Repeat("│  ", len(display))
		b.WriteString(prefix + "...\n")
		fmt.Fprintf(&b, "%s(%d more layers not shown)\n", prefix, len(layers)-len(display))
	}

	return b.String()
}

func (dag *OperationDAG) formatASCIINode(opID string) string {
	node := dag.Nodes[opID]

	var b strings.Builder
	fmt.Fprintf(&b, "%s [%s] '%s'", opID, node.Operation.Op().String(), node.Operation.Char())

	if len(node.Parents) > 1 {

		ids := strings.Join(utils.Map(node.Parents, func(lid cvrdt.LogicalId) string { return lid.String() }), ", ")

		fmt.Fprintf(&b, " [%d parents (%s)]", len(node.Parents), ids)
	}

	children := dag.Edges[opID]
	if len(children) > 0 {
		b.WriteString(" → ")
		b.WriteString(strings.Join(children, ", "))
	}

	return b.String()
}
