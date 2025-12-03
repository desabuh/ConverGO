package utils

// An entity with the ability to merge itself and return a new merged self
type Mergeable[T any] interface {
	Merge(state T) T
}

// Provide a point-in-time rappresentation of an underlying state, also return true if the internal state was updated from the last snapshot
type SnapshotView[V any] interface {
	Snapshot() (V, bool)
}

// An interface for an object to provide its state and update it
type StateStore[X any] interface {
	UpdateState(data X) error
	GetState() X
}
