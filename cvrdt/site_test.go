package cvrdt

import (
	"testing"

	"github.com/desabuh/convergo/utils"
	"github.com/stretchr/testify/assert"
)

func TestConcurrentInsertCRDT(t *testing.T) {
	const SITE_ID_1 = "1"
	const SITE_ID_2 = "2"

	const INSERT_AT_START = 0

	const INSERT_1 = "Hi "
	const INSERT_2 = "User"

	var data SnapshotCvrdt[CvRDTState, string] = NewWootCvrdtWithView(SITE_ID_1)

	var localOp LocalOperation = LocalOperation{
		opType:  Insertion,
		pos:     INSERT_AT_START,
		content: INSERT_1,
		id:      SITE_ID_1,
	}

	err := data.UpdateState(GetNewStateFromOp(localOp))

	assert.Nil(t, err, "Update State in "+SITE_ID_1+" should be successfull")

	assert.Equal(t, INSERT_1, utils.First(data.Snapshot()))

	var data2 SnapshotCvrdt[CvRDTState, string] = NewWootCvrdtWithView(SITE_ID_2)

	var localOp2 LocalOperation = LocalOperation{
		opType:  Insertion,
		pos:     INSERT_AT_START,
		content: INSERT_2,
		id:      SITE_ID_2,
	}

	err = data2.UpdateState(GetNewStateFromOp(localOp2))

	assert.Nil(t, err, "Update State in "+SITE_ID_2+" should be successfull")

	assert.Equal(t, INSERT_2, utils.First(data2.Snapshot()))

	err = data.UpdateState(data2.GetState())
	assert.Nil(t, err, "Update State from "+SITE_ID_2+" into "+SITE_ID_1+" should be successfull")

	assert.Equal(t, utils.First(data.Snapshot()), INSERT_1+INSERT_2)

	err = data2.UpdateState(data.GetState())
	assert.Nil(t, err, "Update State from "+SITE_ID_1+" into "+SITE_ID_2+" should be successfull")

	assert.Equal(t, data.GetState(), data2.GetState())
	assert.Equal(t, utils.First(data.Snapshot()), utils.First(data2.Snapshot()))

}

func TestDuplicateInsert(t *testing.T) {
	const SITE_ID_1 = "1"
	const INSERT_AT_START = 0
	const INSERT = "Test"

	var data SnapshotCvrdt[CvRDTState, string] = NewWootCvrdtWithView(SITE_ID_1)

	var localOpInsert LocalOperation = LocalOperation{
		opType:  Insertion,
		pos:     INSERT_AT_START,
		content: INSERT,
		id:      SITE_ID_1,
	}

	err := data.UpdateState(GetNewStateFromOp(localOpInsert))
	assert.Nil(t, err, "Update State in "+SITE_ID_1+" should be successful")

	data.UpdateState(data.GetState())

	assert.Equal(t, utils.First(data.Snapshot()), INSERT, "Duplicate Operation should not be repeated")

}

func TestConcurrentInsertDeleteCRDT(t *testing.T) {
	const SITE_ID_1 = "1"
	const SITE_ID_2 = "2"

	const INSERT_AT_START = 0
	const DELETE_AT_START = 0

	const INSERT_1 = "Hello"
	const INITIAL_EXPECTED_STATE = ""
	const FINAL_EXPECTED_STATE_AFTER_INSERT = INSERT_1
	const FINAL_EXPECTED_STATE_AFTER_DELETE = INITIAL_EXPECTED_STATE

	var data1 SnapshotCvrdt[CvRDTState, string] = NewWootCvrdtWithView(SITE_ID_1)
	var data2 SnapshotCvrdt[CvRDTState, string] = NewWootCvrdtWithView(SITE_ID_2)

	var localOpInsert LocalOperation = LocalOperation{
		opType:  Insertion,
		pos:     INSERT_AT_START,
		content: INSERT_1,
		id:      SITE_ID_1,
	}

	err := data1.UpdateState(GetNewStateFromOp(localOpInsert))
	assert.Nil(t, err, "Update State in "+SITE_ID_1+" should be successful")
	assert.Equal(t, utils.First(data1.Snapshot()), FINAL_EXPECTED_STATE_AFTER_INSERT)

	err = data2.UpdateState(data1.GetState())
	assert.Nil(t, err, "Update State from "+SITE_ID_1+" into "+SITE_ID_2+" should be successful")
	assert.Equal(t, utils.First(data2.Snapshot()), FINAL_EXPECTED_STATE_AFTER_INSERT)

	var localOpDelete LocalOperation = LocalOperation{
		opType: Deletion,
		pos:    DELETE_AT_START,
		id:     SITE_ID_1,
	}

	err = data1.UpdateState(GetNewStateFromOp(localOpDelete))
	assert.Nil(t, err, "Update State in "+SITE_ID_1+" should be successful")
	assert.Equal(t, utils.First(data1.Snapshot()), FINAL_EXPECTED_STATE_AFTER_DELETE)

	err = data2.UpdateState(data1.GetState())
	assert.Nil(t, err, "Update State from "+SITE_ID_1+" into "+SITE_ID_2+" should be successful")
	assert.Equal(t, utils.First(data2.Snapshot()), FINAL_EXPECTED_STATE_AFTER_DELETE)

	assert.Equal(t, data1.GetState(), data2.GetState())
	assert.Equal(t, utils.First(data1.Snapshot()), string(utils.First(data2.Snapshot())))
}

func TestComputeDependandInsertPendings(t *testing.T) {
	const SITE_ID_1 = "1"
	const SITE_ID_2 = "2"

	const BEFORE_STR = "First part "
	const AFTER_STR = "Second part"
	const EMPTY = ""

	var data1 SnapshotCvrdt[CvRDTState, string] = NewWootCvrdtWithView(SITE_ID_1)

	var localOpInsert LocalOperation = LocalOperation{
		opType:  Insertion,
		pos:     0,
		content: BEFORE_STR,
		id:      SITE_ID_1,
	}

	err := data1.UpdateState(GetNewStateFromOp(localOpInsert))
	assert.Nil(t, err, "Update State for: "+BEFORE_STR+" in "+SITE_ID_1+" should be successful")

	localOpInsert = LocalOperation{
		opType:  Insertion,
		pos:     1,
		content: AFTER_STR,
		id:      SITE_ID_1,
	}

	err = data1.UpdateState(GetNewStateFromOp(localOpInsert))
	assert.Nil(t, err, "Update State for: "+AFTER_STR+" in "+SITE_ID_1+" should be successful")

	assert.Equal(t, utils.First(data1.Snapshot()), BEFORE_STR+AFTER_STR)

	var data2 SnapshotCvrdt[CvRDTState, string] = NewWootCvrdtWithView(SITE_ID_2)

	var firstSiteState CvRDTState = data1.GetState()

	var firstOp, secondOp CRDTOperation
	if len(firstSiteState) >= 2 {
		firstOp = firstSiteState[0]
		secondOp = firstSiteState[1]
	}

	err = data2.UpdateState(GetNewStateFromOp(secondOp))
	assert.Nil(t, err, "Update State for: "+AFTER_STR+" in "+SITE_ID_2+" should be successful")

	assert.Equal(t, utils.First(data2.Snapshot()), EMPTY, "Out of order states should not be immediately added to the CRDT")

	err = data2.UpdateState(GetNewStateFromOp(firstOp))
	assert.Nil(t, err, "Update State for: "+BEFORE_STR+" in "+SITE_ID_2+" should be successful")

	assert.Equal(t, utils.First(data2.Snapshot()), BEFORE_STR+AFTER_STR, "Pendings states should be added after this")

}
