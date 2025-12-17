package cvrdt

import "fmt"

type OpType int

const (
	Insertion OpType = iota
	Deletion
)

type CRDTOperation interface {
	ID() string
	Op() OpType
	Char() string
	// Compare returns -1 if this operation is less than other, 0 if equal, 1 if greater
	// Comparison is based on ID first, then operation type
	Compare(other CRDTOperation) int
}

type WootOperation struct {
	character WCharacter
	opType    OpType
}

func (w WootOperation) ID() string {
	return w.character.id.siteId + "_" + fmt.Sprint(w.character.id.clockValue)
}

func (w WootOperation) Op() OpType {
	return w.opType
}

func (w WootOperation) Char() string {
	return w.character.alphaValue
}

func (w WootOperation) Compare(other CRDTOperation) int {
	thisID := w.ID()
	otherID := other.ID()

	if thisID < otherID {
		return -1
	} else if thisID > otherID {
		return 1
	}

	// If IDs are equal, compare by operation type
	if w.opType < other.Op() {
		return -1
	} else if w.opType > other.Op() {
		return 1
	}

	return 0
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

func (o LocalOperation) ID() string {
	return o.id
}

func (w LocalOperation) Op() OpType {
	return w.opType
}

func (w LocalOperation) Char() string {
	return w.content
}

func (w LocalOperation) Compare(other CRDTOperation) int {
	// First compare by ID
	thisID := w.ID()
	otherID := other.ID()

	if thisID < otherID {
		return -1
	} else if thisID > otherID {
		return 1
	}

	// If IDs are equal, compare by operation type
	if w.opType < other.Op() {
		return -1
	} else if w.opType > other.Op() {
		return 1
	}

	return 0
}
