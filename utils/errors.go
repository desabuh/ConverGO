package utils

// ErrPendingState indicates that an operation could not be completed because
// the target resource or process is currently in a pending state but don't imply a failure in the execution
type ErrPendingState struct {
	Msg string
}

func (e ErrPendingState) Error() string { return e.Msg }
