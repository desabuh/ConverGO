package history

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/desabuh/convergo/cvrdt"
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

// GetCausalityRelation determines the relationship between two operations
func (dag *OperationDAG) GetCausalityRelation(op1Id, op2Id string) CausalityRelation {
	if op1Id == op2Id {
		return Same
	}
	if dag.HasPath(op1Id, op2Id) {
		return HappenedBefore
	}
	if dag.HasPath(op2Id, op1Id) {
		return HappenedAfter
	}
	return Concurrent
}

// CausalityRelation describes the relationship between two operations
type CausalityRelation int

const (
	Same CausalityRelation = iota
	HappenedBefore
	HappenedAfter
	Concurrent
)

func (cr CausalityRelation) String() string {
	switch cr {
	case Same:
		return "same"
	case HappenedBefore:
		return "happened-before"
	case HappenedAfter:
		return "happened-after"
	case Concurrent:
		return "concurrent"
	default:
		return "unknown"
	}
}

// GetTopologicalLayers groups operations into topological (causal layers) for visualization
// Layer 0: operations with no dependencies
// Layer i: operations that depend only on layers 0..i-1
// func (dag *OperationDAG) GetTopologicalLayers() [][]cvrdt.LogicalId {
// 	var layers [][]cvrdt.LogicalId

// 	for _, n := range dag.Nodes {
// 		layer := n.Operation.OpId().GetClock()

// 		for len(layers) <= layer {
// 			layers = append(layers, []cvrdt.LogicalId{})
// 		}

// 		layers[layer] = append(layers[layer], n.Operation.OpId())
// 	}

// 	return layers
// }

func (dag *OperationDAG) GetTopologicalLayers() [][]cvrdt.LogicalId {
	layers := [][]cvrdt.LogicalId{}
	assigned := make(map[string]int) // opId -> layer number

	// Keep assigning layers until all nodes are assigned
	for len(assigned) < len(dag.Nodes) {
		currentLayer := []cvrdt.LogicalId{}

		for opId, node := range dag.Nodes {
			if _, alreadyAssigned := assigned[opId]; alreadyAssigned {
				continue
			}

			// Check if all parents are assigned to previous layers
			canAssign := true
			maxParentLayer := -1

			for _, parentId := range node.Parents {
				parentIdStr := parentId.String()

				if parentLayer, isAssigned := assigned[parentIdStr]; isAssigned {
					if parentLayer > maxParentLayer {
						maxParentLayer = parentLayer
					}
				} else {
					// Parent not yet assigned, can't assign this node yet
					canAssign = false
					break
				}
			}

			// Node can be assigned to layer (maxParentLayer + 1)
			if canAssign {
				currentLayer = append(currentLayer, node.Operation.OpId())
			}
		}

		if len(currentLayer) > 0 {
			// Sort layer by Lamport clock for deterministic display
			sort.Slice(currentLayer, func(i, j int) bool {
				return currentLayer[i].Less(currentLayer[j])
			})

			layers = append(layers, currentLayer)

			// Mark as assigned to this layer
			for _, opId := range currentLayer {
				assigned[opId.String()] = len(layers) - 1
			}
		} else {
			// Safety: break if we can't make progress (shouldn't happen with valid DAG)
			break
		}
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

// ToASCIITree returns an ASCII tree representation of the DAG
// maxDepth limits the nesting depth (e.g., 15) to prevent overwhelming output
// If maxDepth is 0, no limit is applied
// Concurrent operations are shown at the same nesting level
// Sequential/dependent operations are shown at deeper nesting levels
func (dag *OperationDAG) ToASCIITree(maxDepth int) string {
	if len(dag.Nodes) == 0 {
		return "Empty DAG\n"
	}

	var result string
	result += fmt.Sprintf("\n┌─ DAG: %d operations, %d causal edges\n", len(dag.Nodes), dag.countEdges())
	result += "│\n"

	// Get causal layers - each layer contains concurrent operations
	layers := dag.GetTopologicalLayers()

	// Check depth limit
	displayLayers := layers
	truncated := false
	if maxDepth > 0 && len(layers) > maxDepth {
		displayLayers = layers[:maxDepth]
		truncated = true
	}

	// Build tree layer by layer
	for layerIdx, layer := range displayLayers {
		isLastLayer := layerIdx == len(displayLayers)-1

		// Sort operations in layer for deterministic output
		sortedOps := make([]string, len(layer))
		for i, opId := range layer {
			sortedOps[i] = opId.String()
		}
		sort.Strings(sortedOps)

		// Determine prefix for this layer
		prefix := strings.Repeat("│  ", layerIdx)

		// Display all operations in this layer at the same nesting level
		for opIdx, opIdStr := range sortedOps {
			isLastInLayer := opIdx == len(sortedOps)-1
			node := dag.Nodes[opIdStr]

			// Determine branch character
			var branch string
			if isLastLayer && isLastInLayer && !truncated {
				branch = "└─"
			} else {
				branch = "├─"
			}

			// Build operation info
			opInfo := fmt.Sprintf("%s [%s]", opIdStr, node.Operation.Op().String()[:3])

			opInfo += fmt.Sprintf(" '%s'", node.Operation.Char())

			// Show parent count if multiple parents (merge point)
			if len(node.Parents) > 1 {
				opInfo += fmt.Sprintf(" [%d parents]", len(node.Parents))
			}

			result += prefix + branch + " " + opInfo

			// Show direct children inline (if any)
			children := dag.Edges[opIdStr]
			if len(children) > 0 {
				// Sort children for deterministic output
				sortedChildren := make([]string, len(children))
				copy(sortedChildren, children)
				sort.Strings(sortedChildren)

				result += " → "
				for childIdx, childId := range sortedChildren {
					if childIdx > 0 {
						result += ", "
					}
					result += childId
				}
			}

			result += "\n"
		}

		// Add separator between layers if not last
		if !isLastLayer {
			result += prefix + "│\n"
		}
	}

	// Show truncation message if depth limit was reached
	if truncated {
		prefix := strings.Repeat("│  ", len(displayLayers))
		result += prefix + "...\n"
		result += prefix + fmt.Sprintf("(%d more layers not shown)\n", len(layers)-len(displayLayers))
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
