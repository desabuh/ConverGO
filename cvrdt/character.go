package cvrdt

type WCharacterId struct {
	siteId     string
	clockValue int
}

const SPECIAL_START_CHAR = "<START>"
const SPECIAL_END_CHAR = "<END>"

const SPECIAL_START_CHAR_CLOCK = 0
const SPECIAL_END_CHAR_CLOCK = MaxValue
const SPECIAL_MISSING_CHAR_CLOCK = -1

func (id WCharacterId) Equals(otherId WCharacterId) bool {
	return (id.siteId == otherId.siteId && id.clockValue == otherId.clockValue) ||
		(id.clockValue == SPECIAL_START_CHAR_CLOCK && otherId.clockValue == SPECIAL_START_CHAR_CLOCK) ||
		(id.clockValue == SPECIAL_END_CHAR_CLOCK && otherId.clockValue == SPECIAL_END_CHAR_CLOCK)
}

type WCharacter struct {
	id         WCharacterId
	alphaValue string
	visible    bool
	previousId WCharacterId
	nextId     WCharacterId
}

func GetNewSpecialStartWChar(siteId string) *WCharacter {
	return &WCharacter{
		id:         WCharacterId{siteId: siteId, clockValue: SPECIAL_START_CHAR_CLOCK},
		alphaValue: SPECIAL_START_CHAR,
		visible:    true,
		previousId: WCharacterId{siteId: siteId, clockValue: SPECIAL_MISSING_CHAR_CLOCK},
		nextId:     WCharacterId{siteId: siteId, clockValue: SPECIAL_END_CHAR_CLOCK},
	}
}

func GetNewSpecialEndWChar(siteId string) *WCharacter {
	return &WCharacter{
		id:         WCharacterId{siteId: siteId, clockValue: SPECIAL_END_CHAR_CLOCK},
		alphaValue: SPECIAL_END_CHAR,
		visible:    true,
		previousId: WCharacterId{siteId: siteId, clockValue: SPECIAL_START_CHAR_CLOCK},
		nextId:     WCharacterId{siteId: siteId, clockValue: SPECIAL_MISSING_CHAR_CLOCK},
	}
}

func (w WCharacter) Less(other WCharacter) bool {
	if w.id.siteId < other.id.siteId {
		return true
	}
	if w.id.siteId == other.id.siteId && w.id.clockValue < other.id.clockValue {
		return true
	}

	return false
}

func (w *WCharacter) IsSpecialWChar() bool {
	if w == nil {
		return false
	}
	return w.alphaValue == SPECIAL_START_CHAR || w.alphaValue == SPECIAL_END_CHAR
}
