package utils

import (
	"io"
	"sync"
)

// singleton exposure for the logger
var Logger = NewMutexLogger()

// a writer-based mutual exclusion logger, it's an easier snippet to manage multiple writing on a Writer in a mutual exclusive way
type MutexLogger struct {
	mu          sync.Mutex
	writerLocks map[io.Writer]*sync.Mutex
}

func NewMutexLogger() *MutexLogger {
	return &MutexLogger{
		writerLocks: make(map[io.Writer]*sync.Mutex),
	}
}

func (l *MutexLogger) getMutex(w io.Writer) *sync.Mutex {
	l.mu.Lock()
	defer l.mu.Unlock()

	m, ok := l.writerLocks[w]
	if !ok {
		m = &sync.Mutex{}
		l.writerLocks[w] = m
	}
	return m
}

func (l *MutexLogger) Log(w io.Writer, fn func(io.Writer)) {
	m := l.getMutex(w)

	m.Lock()
	defer m.Unlock()

	fn(w)
}
