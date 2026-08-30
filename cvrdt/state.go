package cvrdt

import (
	"github.com/desabuh/convergo/utils"
)

// Represents a state-based Convergent Replicated Data Type whose  internal state X supports monotonic merging
type Cvrdt[X utils.Mergeable[X]] interface {
	utils.StateStore[X]
}

// A decorator interface to provide a simpler view rappresentation of the complex internal crdt state in the form of an up-to-date snapshot
type SnapshotCvrdt[X utils.Mergeable[X], R any] interface {
	Cvrdt[X]
	utils.SnapshotView[R]
}

type CvRDTState []CRDTOperation

// O(log(n))
func (w CvRDTState) binarySearch(op CRDTOperation) (int, bool) {
	left, right := 0, len(w)

	for left < right {
		mid := left + (right-left)/2
		cmp := w[mid].CompareOperation(op)

		if cmp == 0 {
			return mid, true
		} else if cmp < 0 {
			left = mid + 1
		} else {
			right = mid
		}
	}

	return left, false
}

func (w CvRDTState) insert(op CRDTOperation) CvRDTState {
	idx, exists := w.binarySearch(op)

	if exists {
		return w
	}

	w = append(w, nil)
	copy(w[idx+1:], w[idx:])
	w[idx] = op

	return w
}

func (w CvRDTState) contains(op CRDTOperation) bool {
	_, exists := w.binarySearch(op)
	return exists
}

func GetNewStateFromOp(ops ...CRDTOperation) CvRDTState {
	stateInsert := make(CvRDTState, 0, len(ops))
	for _, op := range ops {
		stateInsert = stateInsert.insert(op)
	}
	return stateInsert
}

func (w CvRDTState) Merge(state CvRDTState) CvRDTState {
	for _, op := range state {
		if !w.contains(op) {
			w = w.insert(op)
		}
	}
	return w
}

func (w CvRDTState) Clone() CvRDTState {
	clonedState := make(CvRDTState, len(w))
	copy(clonedState, w)
	return clonedState
}

// A state based CRDT wrapping WOOT operations in a state. It also implement Snapshot[string] to expose a simpler textual rappresentation
// of a Woot site without the need to manually reconstruct it from all operations
type WootCvrdt struct {
	site            Site
	state           CvRDTState
	isStateUpToDate bool
}

func NewWootCvrdtWithView(siteId string) *WootCvrdt {

	return &WootCvrdt{
		site:            *NewSite(siteId),
		state:           make(CvRDTState, 0),
		isStateUpToDate: false,
	}
}

func (w *WootCvrdt) UpdateState(state CvRDTState) error {

	var newState CvRDTState = make(CvRDTState, 0)

	for _, op := range state {
		if !w.state.contains(op) {
			resOp, err := w.site.ComputeOp(op)

			_, ok := err.(*utils.ErrPendingState)

			if err != nil && !ok {
				return err
			}

			newState = newState.insert(resOp)
		}
	}

	w.state = w.state.Merge(newState)

	if len(newState) > 0 {
		w.isStateUpToDate = false
	}

	return nil

}

func (w *WootCvrdt) GetState() CvRDTState {
	return w.state
}

func (w *WootCvrdt) Snapshot() (string, bool) {

	temp := w.isStateUpToDate
	w.isStateUpToDate = true

	return w.site.GetCurrentData(), temp
}
