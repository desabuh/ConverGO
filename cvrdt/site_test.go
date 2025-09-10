package cvrdt

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateNewSite(t *testing.T) {
	const NUM_DEFAULT_CHARS = 2
	const DEFAULT_CLOCK_VALUE = 0
	const SITE_ID = "1"
	const EMPTY_PENDING_OP = 0

	site := NewSite(SITE_ID)

	assert.True(t, site.siteId == SITE_ID, "The site id should be "+SITE_ID)

	assert.True(t, site.characters.GetLen() == NUM_DEFAULT_CHARS, "The site should have by default a start and end WChar")

	startWChar := site.characters.sequence[0]
	endWChar := site.characters.sequence[1]

	assert.True(t, startWChar.alphaValue == SPECIAL_START_CHAR && endWChar.alphaValue == SPECIAL_END_CHAR, "Start and End Wchar use special Char "+SPECIAL_START_CHAR+" and "+SPECIAL_END_CHAR)

	startWcharId := startWChar.id
	endWcharId := endWChar.id

	assert.False(t, startWcharId.Equals(endWcharId), "Default start and end char should be different")

	assert.True(t, startWcharId.siteId == endWcharId.siteId, "Though charId are different their site id portion should be the same")

	assert.True(t, site.clock.Value() == DEFAULT_CLOCK_VALUE, "At the moment of creation clock value should be "+fmt.Sprint(DEFAULT_CLOCK_VALUE))

	assert.True(t, len(site.pendingOpsCh) == EMPTY_PENDING_OP, "At the moment of creation there should be no pending operations")

}

func TestLocalInsertDelete(t *testing.T) {
	const SITE_ID = "1"
	const ELEMENT_INSERT = "A"
	const INSERT_OP = Insertion
	const DELETE_OP = Deletion
	const INSERT_AT_START = 0
	const DELETE_AT_FIRST = 0

	const ORIGINAL_STATE = SPECIAL_START_CHAR + SPECIAL_END_CHAR
	const INSERT_STATE = SPECIAL_START_CHAR + ELEMENT_INSERT + SPECIAL_END_CHAR

	site := NewSite(SITE_ID)

	assert.Equal(t, string(site.GetCurrentData()), ORIGINAL_STATE)

	op, err := site.GenerateIns(INSERT_AT_START, ELEMENT_INSERT)

	assert.Nil(t, err, "Insertion at start should be successfull")

	assert.True(t, op.character.alphaValue == ELEMENT_INSERT, "Operation returned from insertion should provide the Wchar generated")

	assert.True(t, op.opType == INSERT_OP, "Result should be "+fmt.Sprint(INSERT_OP))

	assert.Equal(t, string(site.GetCurrentData()), INSERT_STATE)

	op, err = site.GenerateDel(DELETE_AT_FIRST)

	assert.Nil(t, err, "Deletion at start should be successfull")

	assert.True(t, op.character.alphaValue == ELEMENT_INSERT, "Operation returned from deletion should provide removed character "+ELEMENT_INSERT)

	assert.True(t, op.opType == DELETE_OP, "Result should be "+fmt.Sprint(DELETE_OP))

	assert.Equal(t, string(site.GetCurrentData()), ORIGINAL_STATE)

	op, err = site.GenerateDel(DELETE_AT_FIRST)

	assert.NotNil(t, err, "Deletion should not be possible if no other Wchar are present")

	assert.Equal(t, string(site.GetCurrentData()), ORIGINAL_STATE)

}
