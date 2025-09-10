package cvrdt

type OpType int

const (
	Insertion OpType = iota
	Deletion
)

type WootOperation struct {
	character WCharacter
	opType    OpType
}
