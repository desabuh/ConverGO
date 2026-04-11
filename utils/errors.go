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
