package file

import (
	"os"
	"time"

	"github.com/desabuh/convergo/utils"
)

// a FileContext[M] over an internal state M exposed in the form of a string (e.g. collaborative texting, concurrent logging...)
type StringFileContext[M any] struct {
	*FileContext[M, string]
}

// describe an abstraction over some context metastate M synchronized with an open stream.
// The exposed state R can o cannot (isStateTracked parameter) be continously observed by polling and dumped into the file
type FileContext[M any, R any] struct {
	domain         string
	localPath      string
	metaState      utils.ObservableState[M, R]
	isStateTracked bool
	poller         StatePollingDumper[R]
}

func GetNewFileContext[M any](domain string, localpath string, metaState utils.ObservableState[M, string], file *os.File, pollingInterval time.Duration) *StringFileContext[M] {
	return &StringFileContext[M]{
		FileContext: &FileContext[M, string]{
			domain:         domain,
			localPath:      localpath,
			metaState:      metaState,
			isStateTracked: false,
			poller: GetFileStringPollingDumper(
				metaState,
				file,
				pollingInterval,
			),
		},
	}

}

func (fc *FileContext[M, R]) TrackState() {
	fc.isStateTracked = true
	go fc.poller.Run()
}

func (fc *FileContext[M, R]) UntrackState() {
	fc.isStateTracked = false
	fc.poller.Stop()
}

func (fc *FileContext[M, R]) UpdateState(internalState M) error {
	return fc.metaState.UpdateState(internalState)
}

func (fc *FileContext[M, R]) GetStateCopy() FileContextInfo[M] {
	return FileContextInfo[M]{
		domainPath:    fc.domain,
		localPath:     fc.localPath,
		readOnlyState: fc.metaState,
	}
}

// a readonly version of the FileContext general info and synchronized readonly state
type FileContextInfo[M any] struct {
	domainPath    string
	localPath     string
	readOnlyState utils.ReadonlyStateStore[M]
}
