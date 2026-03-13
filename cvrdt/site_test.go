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

func TestThreeSiteConcurrentConvergence(t *testing.T) {
	const SITE_ID_1 = "1"
	const SITE_ID_2 = "2"
	const SITE_ID_3 = "3"

	const INSERT_1 = "A"
	const INSERT_2 = "B"
	const INSERT_3 = "C"

	// Create three independent sites
	var site1 SnapshotCvrdt[CvRDTState, string] = NewWootCvrdtWithView(SITE_ID_1)
	var site2 SnapshotCvrdt[CvRDTState, string] = NewWootCvrdtWithView(SITE_ID_2)
	var site3 SnapshotCvrdt[CvRDTState, string] = NewWootCvrdtWithView(SITE_ID_3)

	// Each site performs a concurrent insertion at position 0
	localOp1 := LocalOperation{
		opType:  Insertion,
		pos:     0,
		content: INSERT_1,
		id:      SITE_ID_1,
	}
	err := site1.UpdateState(GetNewStateFromOp(localOp1))
	assert.Nil(t, err, "Site 1 insertion should succeed")

	localOp2 := LocalOperation{
		opType:  Insertion,
		pos:     0,
		content: INSERT_2,
		id:      SITE_ID_2,
	}
	err = site2.UpdateState(GetNewStateFromOp(localOp2))
	assert.Nil(t, err, "Site 2 insertion should succeed")

	localOp3 := LocalOperation{
		opType:  Insertion,
		pos:     0,
		content: INSERT_3,
		id:      SITE_ID_3,
	}
	err = site3.UpdateState(GetNewStateFromOp(localOp3))
	assert.Nil(t, err, "Site 3 insertion should succeed")

	// Verify each site has only its own operation
	assert.Equal(t, INSERT_1, utils.First(site1.Snapshot()), "Site 1 should have A")
	assert.Equal(t, INSERT_2, utils.First(site2.Snapshot()), "Site 2 should have B")
	assert.Equal(t, INSERT_3, utils.First(site3.Snapshot()), "Site 3 should have C")

	// Now merge all states: Site 1 receives from Site 2 and Site 3
	err = site1.UpdateState(site2.GetState())
	assert.Nil(t, err, "Site 1 should merge Site 2 state")

	err = site1.UpdateState(site3.GetState())
	assert.Nil(t, err, "Site 1 should merge Site 3 state")

	// Site 2 receives from Site 1 and Site 3
	err = site2.UpdateState(site1.GetState())
	assert.Nil(t, err, "Site 2 should merge Site 1 state")

	err = site2.UpdateState(site3.GetState())
	assert.Nil(t, err, "Site 2 should merge Site 3 state")

	// Site 3 receives from Site 1 and Site 2
	err = site3.UpdateState(site1.GetState())
	assert.Nil(t, err, "Site 3 should merge Site 1 state")

	err = site3.UpdateState(site2.GetState())
	assert.Nil(t, err, "Site 3 should merge Site 2 state")

	// All sites should converge to the same state
	state1 := site1.GetState()
	state2 := site2.GetState()
	state3 := site3.GetState()

	// Verify all states have the same length
	assert.Equal(t, len(state1), len(state2), "Site 1 and Site 2 should have same number of operations")
	assert.Equal(t, len(state1), len(state3), "Site 1 and Site 3 should have same number of operations")
	assert.Equal(t, 3, len(state1), "All sites should have 3 operations")

	// Verify operations are in the same order (by clock, then siteId)
	for i := 0; i < len(state1); i++ {
		assert.Equal(t, state1[i].OpId().String(), state2[i].OpId().String(), "Operation %d should be the same in Site 1 and Site 2", i)
		assert.Equal(t, state1[i].OpId().String(), state3[i].OpId().String(), "Operation %d should be the same in Site 1 and Site 3", i)
		assert.Equal(t, state1[i].Op(), state2[i].Op(), "Operation %d type should be the same in Site 1 and Site 2", i)
		assert.Equal(t, state1[i].Op(), state3[i].Op(), "Operation %d type should be the same in Site 1 and Site 3", i)
	}

	// All concurrent operations have clock=1, so they should be ordered by siteId
	// Expected order: "1_1" (A), "2_1" (B), "3_1" (C)
	assert.Equal(t, SITE_ID_1+"_1", state1[0].OpId().String(), "First operation should be from Site 1")
	assert.Equal(t, SITE_ID_2+"_1", state1[1].OpId().String(), "Second operation should be from Site 2")
	assert.Equal(t, SITE_ID_3+"_1", state1[2].OpId().String(), "Third operation should be from Site 3")

	// Verify all sites have the same visible content
	content1 := utils.First(site1.Snapshot())
	content2 := utils.First(site2.Snapshot())
	content3 := utils.First(site3.Snapshot())

	assert.Equal(t, content1, content2, "Site 1 and Site 2 should have same content")
	assert.Equal(t, content1, content3, "Site 1 and Site 3 should have same content")

	// The content should be deterministic based on site IDs (lexicographic ordering)
	// With site IDs "1", "2", "3" and concurrent insertions, the WOOT algorithm
	// should produce consistent ordering
	t.Logf("Converged content: '%s'", content1)
	t.Logf("Site 1 state: %+v", state1)
}

func TestThreeSiteComplexConcurrentOperations(t *testing.T) {
	const SITE_ID_1 = "1"
	const SITE_ID_2 = "2"
	const SITE_ID_3 = "3"

	// Create three independent sites
	var site1 SnapshotCvrdt[CvRDTState, string] = NewWootCvrdtWithView(SITE_ID_1)
	var site2 SnapshotCvrdt[CvRDTState, string] = NewWootCvrdtWithView(SITE_ID_2)
	var site3 SnapshotCvrdt[CvRDTState, string] = NewWootCvrdtWithView(SITE_ID_3)

	// Site 1: Inserts "Hello"
	err := site1.UpdateState(GetNewStateFromOp(
		LocalOperation{opType: Insertion, pos: 0, content: "Hello", id: SITE_ID_1},
	))
	assert.Nil(t, err)

	// Site 2: Inserts "World"
	err = site2.UpdateState(GetNewStateFromOp(
		LocalOperation{opType: Insertion, pos: 0, content: "World", id: SITE_ID_2},
	))
	assert.Nil(t, err)

	// Site 3: Inserts "!"
	err = site3.UpdateState(GetNewStateFromOp(
		LocalOperation{opType: Insertion, pos: 0, content: "!", id: SITE_ID_3},
	))
	assert.Nil(t, err)

	// Initial states
	assert.Equal(t, "Hello", utils.First(site1.Snapshot()))
	assert.Equal(t, "World", utils.First(site2.Snapshot()))
	assert.Equal(t, "!", utils.First(site3.Snapshot()))

	// First round of merging: Site 1 <-> Site 2
	err = site1.UpdateState(site2.GetState())
	assert.Nil(t, err)
	err = site2.UpdateState(site1.GetState())
	assert.Nil(t, err)

	// Site 1 and Site 2 should now have the same state
	assert.Equal(t, utils.First(site1.Snapshot()), utils.First(site2.Snapshot()))

	// Site 3 adds another character
	err = site3.UpdateState(GetNewStateFromOp(
		LocalOperation{opType: Insertion, pos: 1, content: "?", id: SITE_ID_3},
	))
	assert.Nil(t, err)

	// Site 1 adds a space
	err = site1.UpdateState(GetNewStateFromOp(
		LocalOperation{opType: Insertion, pos: 1, content: " ", id: SITE_ID_1},
	))
	assert.Nil(t, err)

	// Final convergence: everyone gets everyone's state
	// Site 1 <- Site 2, Site 3
	err = site1.UpdateState(site2.GetState())
	assert.Nil(t, err)
	err = site1.UpdateState(site3.GetState())
	assert.Nil(t, err)

	// Site 2 <- Site 1, Site 3
	err = site2.UpdateState(site1.GetState())
	assert.Nil(t, err)
	err = site2.UpdateState(site3.GetState())
	assert.Nil(t, err)

	// Site 3 <- Site 1, Site 2
	err = site3.UpdateState(site1.GetState())
	assert.Nil(t, err)
	err = site3.UpdateState(site2.GetState())
	assert.Nil(t, err)

	// Verify convergence
	state1 := site1.GetState()
	state2 := site2.GetState()
	state3 := site3.GetState()

	assert.Equal(t, len(state1), len(state2), "Site 1 and Site 2 should have same number of operations")
	assert.Equal(t, len(state1), len(state3), "Site 1 and Site 3 should have same number of operations")

	// Verify operations are identical and in the same order
	for i := 0; i < len(state1); i++ {
		assert.Equal(t, state1[i].OpId().String(), state2[i].OpId().String(), "Operation %d ID should match between Site 1 and Site 2", i)
		assert.Equal(t, state1[i].OpId().String(), state3[i].OpId().String(), "Operation %d ID should match between Site 1 and Site 3", i)
	}

	// Verify all sites have the same visible content
	content1 := utils.First(site1.Snapshot())
	content2 := utils.First(site2.Snapshot())
	content3 := utils.First(site3.Snapshot())

	assert.Equal(t, content1, content2, "Site 1 and Site 2 content should match")
	assert.Equal(t, content1, content3, "Site 1 and Site 3 content should match")

	t.Logf("Final converged content: '%s'", content1)
}

func TestThreeSiteWithDeletions(t *testing.T) {
	const SITE_ID_1 = "1"
	const SITE_ID_2 = "2"
	const SITE_ID_3 = "3"

	// Create three independent sites
	var site1 SnapshotCvrdt[CvRDTState, string] = NewWootCvrdtWithView(SITE_ID_1)
	var site2 SnapshotCvrdt[CvRDTState, string] = NewWootCvrdtWithView(SITE_ID_2)
	var site3 SnapshotCvrdt[CvRDTState, string] = NewWootCvrdtWithView(SITE_ID_3)

	// Site 1: Creates initial text "ABC" sequentially
	err := site1.UpdateState(GetNewStateFromOp(
		LocalOperation{opType: Insertion, pos: 0, content: "A", id: SITE_ID_1},
	))
	assert.Nil(t, err)

	err = site1.UpdateState(GetNewStateFromOp(
		LocalOperation{opType: Insertion, pos: 1, content: "B", id: SITE_ID_1},
	))
	assert.Nil(t, err)

	err = site1.UpdateState(GetNewStateFromOp(
		LocalOperation{opType: Insertion, pos: 2, content: "C", id: SITE_ID_1},
	))
	assert.Nil(t, err)
	assert.Equal(t, "ABC", utils.First(site1.Snapshot()))

	// Replicate to Site 2 and Site 3
	err = site2.UpdateState(site1.GetState())
	assert.Nil(t, err)
	err = site3.UpdateState(site1.GetState())
	assert.Nil(t, err)

	assert.Equal(t, "ABC", utils.First(site2.Snapshot()))
	assert.Equal(t, "ABC", utils.First(site3.Snapshot()))

	// Concurrent deletions:
	// Site 1 deletes "B" (position 1)
	err = site1.UpdateState(GetNewStateFromOp(
		LocalOperation{opType: Deletion, pos: 1, id: SITE_ID_1},
	))
	assert.Nil(t, err)
	assert.Equal(t, "AC", utils.First(site1.Snapshot()))

	// Site 2 deletes "A" (position 0)
	err = site2.UpdateState(GetNewStateFromOp(
		LocalOperation{opType: Deletion, pos: 0, id: SITE_ID_2},
	))
	assert.Nil(t, err)
	assert.Equal(t, "BC", utils.First(site2.Snapshot()))

	// Site 3 deletes "C" (position 2)
	err = site3.UpdateState(GetNewStateFromOp(
		LocalOperation{opType: Deletion, pos: 2, id: SITE_ID_3},
	))
	assert.Nil(t, err)
	assert.Equal(t, "AB", utils.First(site3.Snapshot()))

	// Merge all states
	err = site1.UpdateState(site2.GetState())
	assert.Nil(t, err)
	err = site1.UpdateState(site3.GetState())
	assert.Nil(t, err)

	err = site2.UpdateState(site1.GetState())
	assert.Nil(t, err)
	err = site2.UpdateState(site3.GetState())
	assert.Nil(t, err)

	err = site3.UpdateState(site1.GetState())
	assert.Nil(t, err)
	err = site3.UpdateState(site2.GetState())
	assert.Nil(t, err)

	// All sites should converge - all three characters should be deleted
	// since each site deleted a different character
	content1 := utils.First(site1.Snapshot())
	content2 := utils.First(site2.Snapshot())
	content3 := utils.First(site3.Snapshot())

	assert.Equal(t, content1, content2, "Site 1 and Site 2 should converge")
	assert.Equal(t, content1, content3, "Site 1 and Site 3 should converge")
	assert.Equal(t, "", content1, "All characters should be deleted")

	// Verify operation count and order
	state1 := site1.GetState()
	state2 := site2.GetState()
	state3 := site3.GetState()

	assert.Equal(t, len(state1), len(state2), "All sites should have same number of operations")
	assert.Equal(t, len(state1), len(state3), "All sites should have same number of operations")
	assert.Equal(t, 6, len(state1), "Should have 3 insertions + 3 deletions")

	// Verify operations are in the same order
	for i := 0; i < len(state1); i++ {
		assert.Equal(t, state1[i].OpId().String(), state2[i].OpId().String(), "Operation %d should be the same", i)
		assert.Equal(t, state1[i].OpId().String(), state3[i].OpId().String(), "Operation %d should be the same", i)
	}

	// Verify specific ordering
	assert.Equal(t, "1_1", state1[0].OpId().String(), "Op 0: Insert A")
	assert.Equal(t, "1_2", state1[1].OpId().String(), "Op 1: Insert B")
	assert.Equal(t, "1_3", state1[2].OpId().String(), "Op 2: Insert C")
	assert.Equal(t, "1_4", state1[3].OpId().String(), "Op 3: Delete B (clock=4, site=1)")
	// With pure vector clocks, each site's clock only reflects its own operations:
	// Site 2 receives Site 1's ops: vector={1:3} => merges to {1:3, 2:0}
	// Site 2 performs Delete A (its 1st local op): increments to {1:3, 2:1}
	// Site 3 receives Site 1's ops: vector={1:3} => merges to {1:3, 3:0}
	// Site 3 performs Delete C (its 1st local op): increments to {1:3, 3:1}
	assert.Equal(t, "2_1", state1[4].OpId().String(), "Op 4: Delete A (clock=1, site=2)")
	assert.Equal(t, "3_1", state1[5].OpId().String(), "Op 5: Delete C (clock=1, site=3)")

	t.Logf("Final converged content: '%s'", content1)
	t.Logf("Operation order:")
	for i, op := range state1 {
		t.Logf("  %d: %s (%s, type=%v)", i, op.OpId().String(), op.Char(), op.Op())
	}
}
