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

	terminationSignal := make(chan struct{})

	go func() {
		select {
		case msgRec := <-msgCh:
			assert.Equal(t, msg, msgRec, "message sent should match with message received")
		case err := <-msgErr:
			t.Errorf("error from channel while trying to receive messages: %v", err)

		}
		close(terminationSignal)
	}()

	err = peer2.Send(ctx, msg)

	assert.Nil(t, err, "Error in sending message")

	<-terminationSignal

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

	cancel()

	<-terminationSignal

}

func TestMultipleChannelReceiveInstance(t *testing.T) {
	peerConn1, peerConn2 := net.Pipe()
	defer peerConn1.Close()
	defer peerConn2.Close()

	peer1, err := NewTCPPeer(peerConn1)
	assert.Nil(t, err, "wrong transport from connection (not tcp)")

	peer2, err := NewTCPPeer(peerConn2)
	assert.Nil(t, err, "wrong transport from connection (not tcp)")

	defer peer1.Close()
	defer peer2.Close()

	ctx1 := context.Background()

	msgCh, msgErr := peer1.Receive(ctx1)

	terminationSignal := make(chan struct{})

	go func() {
		select {
		case <-msgCh:
		case <-msgErr:
		}
		//when second receive is called the old channels are closed

		close(terminationSignal)
	}()

	ctx2, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, _ = peer1.Receive(ctx2)

	<-terminationSignal

}

func TestMultipleCtxReceiveInstance(t *testing.T) {
	const TOTAL_TIME = 1 * time.Second

	peerConn1, peerConn2 := net.Pipe()

	peer1, err := NewTCPPeer(peerConn1)
	assert.Nil(t, err, "wrong transport from connection (not tcp)")

	peer2, err := NewTCPPeer(peerConn2)
	assert.Nil(t, err, "wrong transport from connection (not tcp)")

	defer peer1.Close()
	defer peer2.Close()

	ctx1, cancel := context.WithTimeout(context.Background(), TOTAL_TIME)
	defer cancel()

	msgC, errC := peer1.Receive(ctx1)

	err = peer2.Send(ctx1, []byte("Test"))

	select {
	case <-msgC:
	case err := <-errC:
		t.Errorf("Error received from peer error channel: %v", err)
	}

	ctx2 := context.Background()

	msgC, errC = peer1.Receive(ctx2)

	select {
	case err := <-errC:
		t.Errorf("Error received from peer error channel: %v", err)
	case <-time.After(2 * TOTAL_TIME): //old ctx should not affect new receive
	}

}
