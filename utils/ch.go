package utils

import "sync"

//a thread safe channel for sending, to avoid concurrent problem of channel close
type SafeChannel[T any] struct {
	mu     sync.RWMutex
	ch     chan T
	closed bool
}

func NewSafeChannel[T any]() SafeChannel[T] {
	return SafeChannel[T]{
		ch: make(chan T),
	}
}

func (s *SafeChannel[T]) Send(v T) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return false
	}

	s.ch <- v
	return true
}

func (s *SafeChannel[T]) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return
	}

	s.closed = true
	close(s.ch)
}

//returning a receive only channel is thread safe
func (s *SafeChannel[T]) GetReceiveOnlyCh() <-chan T {
	return s.ch
}
