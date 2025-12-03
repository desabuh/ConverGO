package file

import (
	"io"
	"os"
	"sync"
	"time"

	"github.com/desabuh/convergo/utils"
)

// A poller that countinuosly dump every interval some data R from a provider into a writer
// Note: the safe concurrent access to the provider should be garanteed by the provider itself
type StatePollingDumper[R any] struct {
	provider   utils.SnapshotView[R]
	writer     io.Writer
	encode     func(R) ([]byte, error)
	onShutDown func() error
	interval   time.Duration
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

func GetCrdtFilePollingDumper[S utils.Mergeable[S]](crdt utils.SnapshotView[string], file *os.File, interval time.Duration) StatePollingDumper[string] {

	return StatePollingDumper[string]{
		provider: crdt,
		writer:   file,
		encode: func(data string) ([]byte, error) { //placeholder will need a change cache
			return []byte(data), nil
		},
		onShutDown: func() error { return file.Close() }, //closure to manage file shutdown
		interval:   interval,
		stopCh:     make(chan struct{}),
	}
}

func (sd *StatePollingDumper[R]) Run() {
	defer sd.wg.Done()
	ticker := time.NewTicker(sd.interval)
	defer ticker.Stop()
	defer sd.onShutDown()

	for {
		select {
		case <-ticker.C:
			var isStateUpToDate bool
			var rappr R
			rappr, isStateUpToDate = sd.provider.Snapshot()

			if isStateUpToDate {
				continue
			}

			b, _ := sd.encode(rappr)

			// Truncate the file before writing to ensure it is overwritten
			if file, ok := sd.writer.(*os.File); ok {
				file.Truncate(0)
				file.Seek(0, 0)
			}

			sd.writer.Write(b)
			sd.writer.Write([]byte("\n"))

		case <-sd.stopCh:
			return
		}
	}
}

func (sd *StatePollingDumper[R]) Stop() {
	close(sd.stopCh)
	sd.wg.Wait()
}
