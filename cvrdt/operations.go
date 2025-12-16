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
