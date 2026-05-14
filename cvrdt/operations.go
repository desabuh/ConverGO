package cvrdt

import (
	"fmt"
)

type OpType int

const (
	Insertion OpType = iota
	Deletion
)

func (ot OpType) String() string {
	switch ot {
	case Insertion:
		return "Insertion"
	case Deletion:
		return "Deletion"
	default:
		return "Unknown"
	}
}

// LogicalId represents a unique identifier with vector clock for causal tracking
// This is the general abstraction used across all CRDT operations
type LogicalId struct {
	SiteId string
	Clock  map[string]int // Vector clock: maps site ID to clock value (includes own clock)
}

func (lid LogicalId) GetClock() int {
	if lid.Clock == nil {
		return 0
	}
	return lid.Clock[lid.SiteId]
}

// Less provides ordering based on vector clock causality
func (lid LogicalId) Less(other LogicalId) bool {

	if lid.happenedBefore(other) {
		return true
	} else if other.happenedBefore(lid) {
		return false
	}

	// Use siteId as deterministic tie-breaker
	return lid.SiteId < other.SiteId
}

// Equals checks if two LogicalIds represent the same operation
func (lid LogicalId) Equals(other LogicalId) bool {
	return lid.SiteId == other.SiteId && lid.GetClock() == other.GetClock()
}

// String returns a string representation of the LogicalId
func (lid LogicalId) String() string {
	return fmt.Sprintf("%s_%d", lid.SiteId, lid.GetClock())
}

// Compare returns -1 if this < other, 0 if equal, 1 if this > other
// Uses the same logic as Less() for consistency
func (lid LogicalId) Compare(other LogicalId) int {
	// Same operation
	if lid.Equals(other) {
		return 0
	}

	if lid.Less(other) {
		return -1
	}

	return 1
}

// happenedBefore checks if this operation causally precedes another
// Returns true if lid's vector clock ≤ other's vector clock (and strictly less in at least one entry)
func (lid LogicalId) happenedBefore(other LogicalId) bool {
	if lid.Clock == nil || len(lid.Clock) == 0 {
		return false
	}

	strictlyLess := false

	// Check if all entries in lid.Clock are ≤ other.Clock
	for site, clock := range lid.Clock {
		otherClock := other.Clock[site] // Returns 0 if site not in other.Clock

		if clock > otherClock {
			// Found an entry where lid > other, so lid did NOT happen-before other
			return false
		}

		if clock < otherClock {
			strictlyLess = true
		}
	}

	// Also check if other has sites that lid doesn't have (with non-zero clocks)
	for site, clock := range other.Clock {
		if _, exists := lid.Clock[site]; !exists && clock > 0 {
			strictlyLess = true
		}
	}

	return strictlyLess
}

// HappenedBefore checks if this operation causally precedes another (public API)
func (lid LogicalId) HappenedBefore(other LogicalId) bool {
	return lid.happenedBefore(other)
}

// IsConcurrent checks if two operations are causally independent
func (lid LogicalId) IsConcurrent(other LogicalId) bool {
	if lid.Equals(other) {
		return false
	}
	return !lid.happenedBefore(other) && !other.happenedBefore(lid)
}

// CRDTOperation is the general interface for all CRDT operations
// Returns LogicalId directly instead of string for type safety and efficiency
type CRDTOperation interface {
	OpId() LogicalId // Returns the operation's unique identifier
	Op() OpType
	Char() string
	// Compare returns -1 if this operation is less than other, 0 if equal, 1 if greater
	Compare(other CRDTOperation) int
}

type WootOperation struct {
	// Operation metadata (for CRDT ordering) - uses LogicalId for consistent ordering
	Id   LogicalId
	Type OpType

	// Character data (for WOOT algorithm) - specific to WOOT
	Character WCharacterData
}

func (w WootOperation) OpId() LogicalId {
	return w.Id
}

func (w WootOperation) Op() OpType {
	return w.Type
}

func (w WootOperation) Char() string {
	return w.Character.AlphaValue
}

func (w WootOperation) Compare(other CRDTOperation) int {
	// Use LogicalId's Compare method - simple and consistent!
	return w.Id.Compare(other.OpId())
}

type LocalOperation struct {
	opType  OpType
	pos     int
	content string

	id string
}

func CreateNewLocalOp(opType OpType, pos int, content string, id string) LocalOperation {
	return LocalOperation{
		opType:  opType,
		pos:     pos,
		content: content,
		id:      id,
	}
}

func (o LocalOperation) OpId() LogicalId {
	// Local operations use site ID as clock 0 temporarily
	// They get a proper LogicalId when converted to WootOperation
	return LogicalId{
		SiteId: o.id,
		Clock:  map[string]int{o.id: 0},
	}
}

func (w LocalOperation) Op() OpType {
	return w.opType
}

func (w LocalOperation) Char() string {
	return w.content
}

// LocalOperation Compare implementation
func (w LocalOperation) Compare(other CRDTOperation) int {
	// Compare using LogicalId
	return w.OpId().Compare(other.OpId())
}
