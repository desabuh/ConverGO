package cvrdt

import (
	"fmt"
	"testing"
	"time"

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

func TestConcurrentInsert(t *testing.T) {
	const SITE_ID_1 = "1"
	const SITE_ID_2 = "2"
	const ELEMENT_INSERT = "A"
	const INITIAL_EXPECTED_STATE = SPECIAL_START_CHAR + SPECIAL_END_CHAR
	const FINAL_EXPECTED_STATE = SPECIAL_START_CHAR + ELEMENT_INSERT + SPECIAL_END_CHAR

	site1 := NewSite(SITE_ID_1)
	site2 := NewSite(SITE_ID_2)

	go site1.ReceptionLoop(100 * time.Millisecond)
	go site2.ReceptionLoop(100 * time.Millisecond)

	assert.Equal(t, string(site1.GetCurrentData()), INITIAL_EXPECTED_STATE, "Site1's initial state should match the expected state")

	op, err := site1.GenerateIns(0, ELEMENT_INSERT)

	assert.Nil(t, err, "Local Insertion at site1 should be successful")
	assert.True(t, op.opType == Insertion, "Operation should be of type Insertion")

	assert.Equal(t, string(site1.GetCurrentData()), FINAL_EXPECTED_STATE, "Site1's state should match the expected state")

	assert.Equal(t, string(site2.GetCurrentData()), INITIAL_EXPECTED_STATE, "Site2's initial state should match the expected state")

	site2.QueueOp(op)

	time.Sleep(1 * time.Second)

	assert.Equal(t, string(site2.GetCurrentData()), FINAL_EXPECTED_STATE, "Site2's initial state should match the expected state")

}

// func TestConcurrentInsertDelete(t *testing.T) {
// 	const SITE_ID_1 = "1"
// 	const SITE_ID_2 = "2"
// 	const ELEMENT_INSERT_1 = "A"
// 	const ELEMENT_INSERT_2 = "B"
// 	//const INSERT_AT_START = 0
// 	const INSERT_AT_END = 1

// 	const EXPECTED_STATE = SPECIAL_START_CHAR + ELEMENT_INSERT_1 + ELEMENT_INSERT_2 + SPECIAL_END_CHAR

// 	site1 := NewSite(SITE_ID_1)
// 	site2 := NewSite(SITE_ID_2)

// 	// Site 1 inserts ELEMENT_INSERT_1 at the start
// 	//op1, err1 := site1.GenerateIns(INSERT_AT_START, ELEMENT_INSERT_1)
// 	//assert.Nil(t, err1, "Insertion at site1 should be successful")

// 	// Site 2 inserts ELEMENT_INSERT_2 at the end
// 	op2, err2 := site2.GenerateIns(INSERT_AT_END, ELEMENT_INSERT_2)
// 	assert.Nil(t, err2, "Insertion at site2 should be successful")

// 	fmt.Printf("op2: %v\n", op2)

// 	//go site2.ReceptionLoop(100 * time.Millisecond)
// 	//site2.QueueOp(op1)

// 	// Apply operations from site2 to site1
// 	go site1.ReceptionLoop(100 * time.Millisecond)
// 	site1.QueueOp(op2)

// 	time.Sleep(1 * time.Second)

// 	// Verify that both sites converge to the same state
// 	assert.Equal(t, string(site1.GetCurrentData()), EXPECTED_STATE, "Site1's state should match the expected state")
// 	//assert.Equal(t, string(site2.GetCurrentData()), EXPECTED_STATE, "Site2's state should match the expected state")
// }
