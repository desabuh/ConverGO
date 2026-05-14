package p2p

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// Peer represents a remote node view. No assumption is made about the peer's nature
// (e.g., connection, RPC). It can be closed and sent byte representations.
// Send is intended to be blocking (after the send the control flow return), this is useful to track message send timeouts for example
// Receive is non-blocking but its behavior depends on effective implementation: e.g. TCPPeer will receive msg or error bytes through a connection,
// gRCP will receive protobuf messages. The only common aspect is that (independently from the implementation) all the data should be supplied through
// a receive only channel (both error and data)
type Peer interface {
	io.Closer
	Send(context.Context, []byte) error
	Receive(context.Context) (<-chan []byte, <-chan error)
	SetPeerInfo(info PeerHostInfo)
	GetPeerInfo() PeerHostInfo
}

const (
	defaultWriteTimeout = 5 * time.Second
	defaultReadTimeout  = 30 * time.Second
)

type TCPPeer struct {
	mu sync.Mutex
	net.Conn

	remotePeerInfo PeerHostInfo

	msgCh chan []byte
	errCh chan error

	recVOnce sync.Once

	RemoteAddress net.Addr
}

func NewTCPPeer(conn net.Conn) (*TCPPeer, error) {
	// if network := conn.RemoteAddr().Network(); network != "tcp" && network != "tcp4" && network != "tcp6" {
	// 	return nil, fmt.Errorf("connection is not TCP: %s", network)
	// }

	if conn == nil {
		return nil, fmt.Errorf("provided connection is not valid")
	}

	return &TCPPeer{
		Conn:          conn,
		RemoteAddress: conn.RemoteAddr(),
	}, nil

}

func (t *TCPPeer) Send(ctx context.Context, bytes []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	var deadline time.Time

	if ctxDeadline, ok := ctx.Deadline(); ok {
		deadline = ctxDeadline
	} else {
		deadline = time.Now().Add(defaultWriteTimeout)
	}

	if err := t.Conn.SetWriteDeadline(deadline); err != nil {
		return err
	}
	defer t.Conn.SetWriteDeadline(time.Time{})

	totalSent := 0
	for totalSent < len(bytes) {

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, err := t.Conn.Write(bytes[totalSent:])
		if err != nil {
			return err
		}
		totalSent += n
	}

	return nil
}

// This method is not idempotent: the first call for this method create a readloop gourutine and return msg and error channel
// Successive calls will simply return the channel
// Important: successive calls have no ownership over the context control so, the internal goroutine can only be stopped through
// explicit closing the connection with the peer or timeout the context owner
func (t *TCPPeer) Receive(ctx context.Context) (<-chan []byte, <-chan error) {
	t.recVOnce.Do(func() {
		t.msgCh = make(chan []byte, 16)
		t.errCh = make(chan error, 1)

		go t.readLoop(ctx)
	})

	return t.msgCh, t.errCh
}

func (t *TCPPeer) readLoop(ctx context.Context) {
	defer close(t.msgCh)
	defer close(t.errCh)

	buf := make([]byte, 1024)

	cancelDone := make(chan struct{})

	go func() {
		select {
		case <-ctx.Done():
			if ctx.Err() == context.Canceled {
				_ = t.Conn.SetReadDeadline(time.Now())
			}
		case <-cancelDone:
		}
	}()

	for {
		n, err := t.Conn.Read(buf)
		if err != nil {
			if _, ok := err.(net.Error); ok {
				select {
				case <-ctx.Done():
					t.errCh <- ctx.Err()
					close(cancelDone)
					return
				default:
					t.errCh <- err
					close(cancelDone)
					return
				}
			}

			if err == io.EOF {
				t.errCh <- err
				close(cancelDone)
				return
			}

			t.errCh <- err
			continue
		}

		msg := make([]byte, n)
		copy(msg, buf[:n])

		select {
		case t.msgCh <- msg:
		case <-ctx.Done():
			t.errCh <- ctx.Err()
			close(cancelDone)
			return
		}
	}
}

func (t *TCPPeer) GetPeerInfo() PeerHostInfo {
	return t.remotePeerInfo
}

func (t *TCPPeer) SetPeerInfo(info PeerHostInfo) {
	t.remotePeerInfo = info
}

func (t *TCPPeer) Close() error {
	return t.Conn.Close()
}
