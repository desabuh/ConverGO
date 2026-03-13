package cvrdt

// WCharacterId is a lightweight identifier for WOOT characters
// Uses only Lamport clock (not full vector clock) for efficient comparison
type WCharacterId struct {
	SiteId string
	Clock  int // Lamport clock value (not vector clock)
}

// Less provides fast Lamport clock ordering for WOOT algorithm
func (wid WCharacterId) Less(other WCharacterId) bool {
	// Compare by Lamport clock first
	if wid.Clock < other.Clock {
		return true
	}
	if wid.Clock > other.Clock {
		return false
	}
	// If clocks are equal, compare by siteId for deterministic ordering
	return wid.SiteId < other.SiteId
}

// Equals checks if two WCharacterIds are equal
func (wid WCharacterId) Equals(other WCharacterId) bool {
	return wid.SiteId == other.SiteId && wid.Clock == other.Clock
}

const SPECIAL_START_CHAR = "<START>"
const SPECIAL_END_CHAR = "<END>"

const SPECIAL_START_CHAR_CLOCK = 0
const SPECIAL_END_CHAR_CLOCK = MaxValue
const SPECIAL_MISSING_CHAR_CLOCK = -1

// WCharacterIdEquals checks equality for WCharacterId with special WOOT boundary handling
func WCharacterIdEquals(id1, id2 WCharacterId) bool {
	// Special handling for WOOT boundary characters
	// Start and End characters are considered equal regardless of site
	if (id1.Clock == SPECIAL_START_CHAR_CLOCK && id2.Clock == SPECIAL_START_CHAR_CLOCK) ||
		(id1.Clock == SPECIAL_END_CHAR_CLOCK && id2.Clock == SPECIAL_END_CHAR_CLOCK) {
		return true
	}

	return id1.Equals(id2)
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
		id:         WCharacterId{SiteId: siteId, Clock: SPECIAL_START_CHAR_CLOCK},
		alphaValue: SPECIAL_START_CHAR,
		visible:    true,
		previousId: WCharacterId{SiteId: siteId, Clock: SPECIAL_MISSING_CHAR_CLOCK},
		nextId:     WCharacterId{SiteId: siteId, Clock: SPECIAL_END_CHAR_CLOCK},
	}
}

func GetNewSpecialEndWChar(siteId string) *WCharacter {
	return &WCharacter{
		id:         WCharacterId{SiteId: siteId, Clock: SPECIAL_END_CHAR_CLOCK},
		alphaValue: SPECIAL_END_CHAR,
		visible:    true,
		previousId: WCharacterId{SiteId: siteId, Clock: SPECIAL_START_CHAR_CLOCK},
		nextId:     WCharacterId{SiteId: siteId, Clock: SPECIAL_MISSING_CHAR_CLOCK},
	}
}

func (w WCharacter) Less(other WCharacter) bool {
	return w.id.Less(other.id)
}

func (w *WCharacter) IsSpecialWChar() bool {
	if w == nil {
		return false
	}
	return w.alphaValue == SPECIAL_START_CHAR || w.alphaValue == SPECIAL_END_CHAR
}
