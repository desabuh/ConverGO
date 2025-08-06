package p2p

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSendContextDeadlineTimeout(t *testing.T) {
	remoteNode, peerConn := net.Pipe()
	defer remoteNode.Close()

	peer, err := NewTCPPeer(peerConn)

	assert.Nil(t, err, "wrong transport from connection (not tcp)")

	defer peer.Close()

	timeout := 100 * time.Millisecond
	delta := 5 * time.Millisecond
	largeByteRapr := 10 << 20

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	largeMsg := make([]byte, largeByteRapr)

	start := time.Now()
	err = peer.Send(ctx, largeMsg)
	elapsed := time.Since(start)

	assert.NotNil(t, err, "Exprected error in peer message send")

	assert.Condition(
		t,
		func() bool { return errors.Is(err, context.DeadlineExceeded) || isTimeoutError(err) },
		"expected context deadline exceeded or timeout error, got: "+err.Error(),
	)

	assert.Condition(
		t,
		func() bool { return elapsed < (timeout * delta) },
		"Send took too long to timeout: "+elapsed.String(),
	)

}

func isTimeoutError(err error) bool {
	netErr, ok := err.(net.Error)
	return ok && netErr.Timeout()
}

func TestSendReceivePeer(t *testing.T) {
	peerConn1, peerConn2 := net.Pipe()

	peer1, err := NewTCPPeer(peerConn1)
	assert.Nil(t, err, "wrong transport from connection (not tcp)")

	peer2, err := NewTCPPeer(peerConn2)
	assert.Nil(t, err, "wrong transport from connection (not tcp)")

	defer peer1.Close()
	defer peer2.Close()

	ctx := context.Background()

	msg := []byte("Test message")

	msgCh, msgErr := peer1.Receive(ctx)

	go func() {
		select {
		case msgRec, ok := <-msgCh:
			if !ok {
				t.Errorf("error with message channel")
			}
			assert.Equal(t, msg, msgRec, "message sent should match with message received")

		case <-msgErr:
			t.Errorf("error from channel while trying to receive messages")
		}
	}()

	err = peer2.Send(ctx, msg)

	assert.Nil(t, err, "Error in sending message")

}

func TestReceiveInterrupted(t *testing.T) {
	peerConn1, peerConn2 := net.Pipe()

	peer1, err := NewTCPPeer(peerConn1)
	assert.Nil(t, err, "wrong transport from connection (not tcp)")

	peer2, err := NewTCPPeer(peerConn2)
	assert.Nil(t, err, "wrong transport from connection (not tcp)")

	defer peer1.Close()
	defer peer2.Close()

	ctx, cancel := context.WithCancel(context.Background())

	msgCh, msgErr := peer1.Receive(ctx)

	terminationSignal := make(chan struct{})

	go func() {
		select {
		case _, ok := <-msgCh:
			assert.False(t, ok, "No message was sent")

		case err := <-msgErr:
			assert.Equal(t, context.Canceled, err, "expected context.Canceled error")
			close(terminationSignal)
		}
	}()

	time.Sleep(500 * time.Millisecond)

	cancel()

	<-terminationSignal

}
