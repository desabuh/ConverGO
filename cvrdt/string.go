package cvrdt

import (
	"fmt"
	"slices"
)

type WString struct {
	sequence []*WCharacter
}

func NewWString(siteId string) *WString {

	head := GetNewSpecialStartWChar(siteId)
	tail := GetNewSpecialEndWChar(siteId)

	return &WString{
		sequence: []*WCharacter{head, tail},
	}

}

func (w *WString) GetById(charId WCharacterId) *WCharacter {
	for i := range w.sequence {
		if WCharacterIdEquals(w.sequence[i].id, charId) {
			return w.sequence[i]
		}
	}
	return nil
}

func (w WString) GetLen() int {
	return len(w.sequence)
}

func (w WString) GetPos(charId WCharacterId) int {
	for pos, el := range w.sequence {
		if WCharacterIdEquals(el.id, charId) {
			return pos
		}
	}
	return -1
}

func (w *WString) Insert(char *WCharacter, position int) error {
	if position < 0 || position > w.GetLen() {
		return fmt.Errorf("index %d out of bound", position)
	}

	w.sequence = slices.Insert(w.sequence, position, char)

	return nil
}

func (w WString) SubSeq(startChar WCharacter, endChar WCharacter) []*WCharacter {
	startPos := w.GetPos(startChar.id)
	endPos := w.GetPos(endChar.id)

	if startPos == -1 || endPos == -1 || startPos >= endPos {
		return nil
	}

	var subSeq []*WCharacter
	for i := startPos + 1; i < endPos; i++ {
		subSeq = append(subSeq, w.sequence[i])
	}

	return subSeq
}

func (w WString) Contains(element WCharacterId) bool {
	return slices.ContainsFunc(w.sequence, func(char *WCharacter) bool {
		return WCharacterIdEquals(char.id, element)
	})
}

func (w *WString) GetIthVisibleValue(occ int, excludeLimiters bool) (*WCharacter, error) {

	if occ < 0 {
		return nil, fmt.Errorf("occurrence index %d out of range", occ)
	}

	var visibleChars []*WCharacter
	for i := range w.sequence {
		if w.sequence[i].visible && !(excludeLimiters && w.sequence[i].IsSpecialWChar()) {
			visibleChars = append(visibleChars, w.sequence[i])
		}
	}

	if len(visibleChars) == 0 {
		return nil, fmt.Errorf("no elements are present to be deleted")
	}

	cappedPos := min(len(visibleChars)-1, occ)

	return visibleChars[cappedPos], nil
}

func (w WString) GetVisibleContent() string {
	var visibleContent string
	for _, char := range w.sequence {
		if char.visible && !char.IsSpecialWChar() {
			visibleContent += char.alphaValue
		}
	}
	return visibleContent
}
