package cvrdt

import (
	"github.com/desabuh/convergo/utils"
)

// An entity with the ability to merge itself and return a new merged self
type Mergeable[T any] interface {
	Merge(state T) T
}

// Provide a point-in-time rappresentation of an underlying state, also return true if the internal state was updated from the last snapshot
type SnapshotView[V any] interface {
	Snapshot() (V, bool)
}

// an interface to provide a state based CRDT for a specific data structure X with the capability to merge
type Cvrdt[X Mergeable[X], R any] interface {
	UpdateState(data X) error
	GetState() X
	SnapshotView[R]
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

type WootCvrdt struct {
	site            Site
	state           CvRDTState
	isStateUpToDate bool
}

func NewWootCvrdt(siteId string) *WootCvrdt {

	return &WootCvrdt{
		site:            *NewSite(siteId),
		state:           make(CvRDTState, 0),
		isStateUpToDate: true,
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
