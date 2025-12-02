package file

import (
	"io"
	"os"
	"sync"
	"time"

	"github.com/desabuh/convergo/cvrdt"
)

// A poller that countinuosly dump every interval some data R from a provider into a writer
// Note: the safe concurrent access to the provider should be garanteed by the provider itself
type StatePollingDumper[R any] struct {
	provider   interface{ GetStateRappr() R }
	writer     io.Writer
	encode     func(R) ([]byte, error)
	onShutDown func() error
	interval   time.Duration
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

func GetCrdtFilePollingDumper[S cvrdt.Mergeable[S]](crdt cvrdt.Cvrdt[S, []byte], writer *os.File, interval time.Duration) StatePollingDumper[[]byte] {

	return StatePollingDumper[[]byte]{
		provider: crdt,
		writer:   writer,
		encode: func(data []byte) ([]byte, error) { //placeholder will need a change cache
			return data, nil
		},
		onShutDown: func() error { return writer.Close() }, //closure to manage file shutdown
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
			rappr := sd.provider.GetStateRappr()

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
