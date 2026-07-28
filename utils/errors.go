package utils

import (
	"fmt"
)

// ErrPendingState indicates that an operation could not be completed because
// the target resource or process is currently in a pending state but don't imply a failure in the execution
type ErrPendingState struct {
	Msg string
}

func (e ErrPendingState) Error() string { return e.Msg }

// An error wrapper to provide underlying error returned in an handshake procedure and at what step
type HandshakeError struct {
	Step   string
	Target any
	Err    error
}

func (e *HandshakeError) Error() string {
	return fmt.Sprintf("handshake with %v address failed at step %s: %v", e.Target, e.Step, e.Err)
}

func (e *HandshakeError) Unwrap() error {
	return e.Err
}

type endpointData interface{ Format() string }

// A generic error to provide info to partial broadcast communication errors
type BroadcastError[T comparable] struct {
	Targets     []T
	FailedIndex int
	Cause       error
}

func (e *BroadcastError[T]) Error() string {
	return fmt.Sprintf(
		"broadcast failed on target %v (send to %v was successfull, send to %v was aborted): %v",
		e.FailedTarget(),
		e.SuccessTargets(),
		e.NotSent(),
		e.Cause,
	)
}

func (e *BroadcastError[T]) Unwrap() error {
	return e.Cause
}

func (e *BroadcastError[T]) FailedTarget() T {
	return e.Targets[e.FailedIndex]
}

func (e *BroadcastError[T]) NotSent() []T {
	return e.Targets[e.FailedIndex:]
}

func (e *BroadcastError[T]) SuccessTargets() []T {
	return e.Targets[:e.FailedIndex]
}
