package utils

// An entity with the ability to merge itself and return a new merged self and to be Clonable
type Mergeable[T any] interface {
	Merge(state T) T
	Clonable[T]
}

// Provide a StateStore that can be also Observed (with a pull approach) and return a rappresentation R of the state S
type ObservableState[S Clonable[S], R any] interface {
	StateStore[S]
	SnapshotView[R]
}

// Provide a point-in-time rappresentation of an underlying state, also return true if the internal state was updated from the last snapshot
type SnapshotView[V any] interface {
	Snapshot() (V, bool)
}

// An interface for an object to provide its state and update it
type StateStore[X Clonable[X]] interface {
	UpdateState(data X) error
	ReadonlyStateStore[X]
}

type ReadonlyStateStore[X Clonable[X]] interface {
	GetState() X
}

// an interface for an object with the ability to deepvopy itself
type Clonable[X any] interface {
	Clone() X
}
