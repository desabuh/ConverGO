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

type CvRDTState map[CRDTOperation]struct{}

func GetNewStateFromOp(ops ...CRDTOperation) CvRDTState {
	stateInsert := make(CvRDTState, len(ops))
	for _, op := range ops {
		stateInsert[op] = struct{}{}
	}
	return stateInsert
}

func (w CvRDTState) Merge(state CvRDTState) CvRDTState {
	for op := range state {
		if _, exists := w[op]; !exists {
			w[op] = struct{}{}
		}
	}
	return w
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

	var newState CvRDTState = make(map[CRDTOperation]struct{})

	for op := range state {
		if _, ok := w.state[op]; !ok {
			resOp, err := w.site.ComputeOp(op)

			_, ok = err.(*utils.ErrPendingState)

			if err != nil && !ok {
				return err
			}

			newState[resOp] = struct{}{}
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

func (w *WootCvrdt) Clone() utils.StateStore[CvRDTState] {
	clonedState := make(CvRDTState, len(w.state))
	for op, value := range w.state {
		clonedState[op] = value
	}

	clonedSite := *NewSite(w.site.siteId)

	return &WootCvrdt{
		site:            clonedSite,
		state:           clonedState,
		isStateUpToDate: false,
	}
}
