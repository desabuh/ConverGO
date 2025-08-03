package p2p

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSynchronousPeer(t *testing.T) {
	remoteNode, peerConn := net.Pipe()

	peer, err := NewTCPPeer(peerConn)

	assert.Nil(t, err, "connection should be tcp based")

	defer peer.Close()

	ctx := context.Background()

	msg := []byte("Test message")

	go func() {
		err = peer.Send(ctx, msg)

		assert.Nil(t, err, "error in peer send message")

		peer.Close()
	}()

	data, err := io.ReadAll(remoteNode)

	assert.Nil(t, err, "error while reading data from connection")

	assert.Equal(t, data, msg, "byte read should match with byte written by remote peer")

}

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
