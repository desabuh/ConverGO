package file

import (
	"io"
	"sync"
	"time"
)

// A poller that countinuosly dump every interval some data R from a provider into a writer
// Note: the safe concurrent access to the provider should be garanteed by the provider itself
type StatePollingDumper[R any] struct {
	provider interface{ GetStateRappr() R }
	writer   io.Writer
	encode   func(R) ([]byte, error)
	interval time.Duration
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

func (sd *StatePollingDumper[R]) Run() {
	defer sd.wg.Done()
	ticker := time.NewTicker(sd.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rappr := sd.provider.GetStateRappr()

			b, _ := sd.encode(rappr)

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
