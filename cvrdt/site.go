package cvrdt

import (
	"fmt"
	"sync"
	"time"
)

type Site struct {
	siteId       string
	clock        LamportClock
	pendingOpsCh chan WootOperation

	mu         sync.Mutex
	characters WString
}

func NewSite(siteId string) *Site {
	return &Site{
		siteId:       siteId,
		clock:        LamportClock{},
		characters:   *NewWString(siteId),
		pendingOpsCh: make(chan WootOperation, 100),
	}
}

func (s *Site) GenerateIns(pos int, str string) (WootOperation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

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
	s.mu.Lock()
	defer s.mu.Unlock()

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

	return targetWChar.IsSpecialWChar() ||
		(op.opType == Deletion && s.characters.Contains(targetWChar.id)) ||
		(s.characters.Contains(targetWChar.previousId) && s.characters.Contains(targetWChar.nextId))

}

func (s *Site) QueueOp(op WootOperation) {
	s.pendingOpsCh <- op
}

func (s *Site) ReceptionLoop(retryTime time.Duration) {
	for op := range s.pendingOpsCh {
		s.mu.Lock()
		if s.IsExecutable(op) {
			if op.opType == Deletion {
				charToRemove := s.characters.GetById(op.character.id)
				s.IntegrateDel(charToRemove)
			} else {
				prev := s.characters.GetById(op.character.previousId)
				next := s.characters.GetById(op.character.nextId)

				s.IntegrateIns(&op.character, prev, next)
			}
		} else {
			//if char does not still exist (delete) or prev-next do not still exist (insert) wait 0.1 sec and try retransmitt on channel
			go func(o WootOperation) {
				time.Sleep(retryTime)
				s.pendingOpsCh <- o
			}(op)
		}
		s.mu.Unlock()
	}
}

func (s *Site) IntegrateIns(wchar *WCharacter, wprev *WCharacter, wnext *WCharacter) {
	subStr := s.characters.SubSeq(*wprev, *wnext) //character between previous and next

	//if no other char are between the prev and next, it'll simply be interted between them
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
	for _, v := range s.characters.GetVisibleContent() {
		fmt.Printf("v: %s\n", string(v))
	}
	return s.characters.GetVisibleContent()
}
