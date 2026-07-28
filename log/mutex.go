package log

import (
	"fmt"
	"io"
	"sync"
)

const (
	FORMAT_PREFIX = "[%s] "
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

func (l *MutexLogger) NlLog(w io.Writer, content string, args ...any) {
	l.Log(w, func(w io.Writer) {
		fmt.Fprintln(w, fmt.Sprintf(content, args...))
	})
}

func (l *MutexLogger) PrefNlog(prefix string, w io.Writer, content string, args ...any) {
	args = append([]any{prefix}, args...)
	l.NlLog(w, FORMAT_PREFIX+content+" ", args...)
}
