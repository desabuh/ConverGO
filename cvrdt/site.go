package cvrdt

import (
	"fmt"

	"github.com/desabuh/convergo/utils"
)

type Site struct {
	siteId string
	clock  LamportClock

	characters WString

	pendingOps []WootOperation
}

func NewSite(siteId string) *Site {
	return &Site{
		siteId:     siteId,
		clock:      LamportClock{},
		characters: *NewWString(siteId),

		pendingOps: make([]WootOperation, 0),
	}
}

func (s *Site) ComputeOp(operation CRDTOperation) (WootOperation, error) {

	switch op := operation.(type) {
	case WootOperation:
		err := s.computeExtOp(op)

		if err == nil {
			s.tryComputePendings()
		}

		return op, err
	case LocalOperation:
		if op.opType == Insertion {
			return s.GenerateIns(op.pos, op.content)
		} else {
			return s.GenerateDel(op.pos)
		}
	default:
		return WootOperation{}, fmt.Errorf("operation provided type was not compatible")
	}

}

func (s *Site) tryComputePendings() {
	for i := 0; i < len(s.pendingOps); i++ {
		oldOp := s.pendingOps[i]
		err := s.computeExtOp(oldOp)

		if err == nil {
			s.pendingOps = append(s.pendingOps[:i], s.pendingOps[i+1:]...)
			s.tryComputePendings()
			break
		}
	}
}

func (s *Site) computeExtOp(op WootOperation) error {
	if s.IsExecutable(op) {
		if op.opType == Deletion {
			charToRemove := s.characters.GetById(op.character.id)
			s.IntegrateDel(charToRemove)
		} else {
			prev := s.characters.GetById(op.character.previousId)
			next := s.characters.GetById(op.character.nextId)

			s.IntegrateIns(&op.character, prev, next)
		}
		return nil
	}

	s.pendingOps = append(s.pendingOps, op)
	return &utils.ErrPendingState{Msg: "Operation with id " + op.ID() + " cannot currently be executed"}
}

func (s *Site) GenerateIns(pos int, str string) (WootOperation, error) {

	newValue := s.clock.Increment()

	prevChar, err := s.characters.GetIthVisibleValue(pos, false)

	if err != nil {
		return WootOperation{}, err
	}

	nextChar, err := s.characters.GetIthVisibleValue(pos+1, false)

	if err != nil {
		return WootOperation{}, err
	}

	wchar := &WCharacter{
		id: WCharacterId{
			s.siteId,
			newValue,
		},
		alphaValue: str,
		visible:    true,
		previousId: prevChar.id,
		nextId:     nextChar.id,
	}

	s.IntegrateIns(wchar, prevChar, nextChar)

	return WootOperation{
		character: *wchar,
		opType:    Insertion,
	}, nil

}

func (s *Site) GenerateDel(pos int) (WootOperation, error) {

	wchar, err := s.characters.GetIthVisibleValue(pos, true)

	if err != nil {
		return WootOperation{}, err
	}

	s.IntegrateDel(wchar)

	return WootOperation{
		character: *wchar,
		opType:    Deletion,
	}, nil
}

func (s *Site) IsExecutable(op WootOperation) bool {
	targetWChar := op.character

	return (op.opType == Deletion && s.characters.Contains(targetWChar.id)) ||
		(s.characters.Contains(targetWChar.previousId) || s.characters.GetById(targetWChar.previousId).IsSpecialWChar()) &&
			(s.characters.Contains(targetWChar.nextId) || s.characters.GetById(targetWChar.nextId).IsSpecialWChar())

}

func (s *Site) IntegrateIns(wchar *WCharacter, wprev *WCharacter, wnext *WCharacter) {

	subStr := s.characters.SubSeq(*wprev, *wnext) //character between previous and next

	//if no other char are between the prev and next, it'll simply be inserted between them
	if len(subStr) == 0 {
		nextPos := s.characters.GetPos(wnext.id)
		s.characters.Insert(wchar, nextPos)
		return
	}

	L := []*WCharacter{wprev}
	for _, di := range subStr {
		posCPDi := s.characters.GetPos(di.previousId)
		posCNDi := s.characters.GetPos(di.nextId)
		posCP := s.characters.GetPos(wprev.id)
		posCN := s.characters.GetPos(wnext.id)

		if posCPDi != -1 && posCNDi != -1 && posCPDi <= posCP && posCN <= posCNDi {
			L = append(L, di)
		}
	}

	L = append(L, wnext)

	i := 1
	for i < len(L)-1 && L[i].Less(*wchar) {
		i++
	}

	s.IntegrateIns(wchar, L[i-1], L[i])
}

func (s *Site) IntegrateDel(wchar *WCharacter) {
	wchar.visible = false
}

func (s *Site) GetCurrentData() []byte {
	return s.characters.GetVisibleContent()
}
