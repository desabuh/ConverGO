package cvrdt

import "fmt"

type OpType int

const (
	Insertion OpType = iota
	Deletion
)

type CRDTOperation interface {
	ID() string
}

type WootOperation struct {
	character WCharacter
	opType    OpType
}

func (w WootOperation) ID() string {
	return w.character.id.siteId + "_" + fmt.Sprint(w.character.id.clockValue)
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
