package file

import (
	"os"
	"time"

	"github.com/desabuh/convergo/utils"
)

// a FileContext[M] over an internal state M exposed in the form of a string (e.g. collaborative texting, concurrent logging...)
type StringFileContext[M utils.Clonable[M]] struct {
	*FileContext[M, string]
}

// describe an abstraction over some observable metastate with state M synchronized with an open stream with an exposed rappresentation R.
// The exposed state R can o cannot (isStateTracked parameter) be continously observed by polling and dumped into the file
// The internal M state should be Clonable for external exposure reasons
type FileContext[M utils.Clonable[M], R any] struct {
	localPath      string
	metaState      utils.ObservableState[M, R]
	isStateTracked bool
	poller         StatePollingDumper[R]
}

func GetNewFileContext[M utils.Clonable[M]](localpath string, metaState utils.ObservableState[M, string], file *os.File, pollingInterval time.Duration) *StringFileContext[M] {
	return &StringFileContext[M]{
		FileContext: &FileContext[M, string]{
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

func (fc *FileContext[M, R]) UntrackState() error {
	fc.isStateTracked = false
	return fc.poller.Stop()
}

func (fc *FileContext[M, R]) UpdateState(internalState M) error {
	return fc.metaState.UpdateState(internalState)
}

func (fc *FileContext[M, R]) GetStateCopy() FileContextInfo[M] {

	return FileContextInfo[M]{
		LocalPath:     fc.localPath,
		ReadOnlyState: fc.metaState.GetState().Clone(),
	}
}

// a readonly version of the FileContext general info and synchronized readonly state
type FileContextInfo[M any] struct {
	LocalPath      string
	IsNewlyCreated bool
	ReadOnlyState  M
}
