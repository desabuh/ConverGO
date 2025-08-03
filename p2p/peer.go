package p2p

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"
)

// Peer represents a remote node view. No assumption is made about the peer's nature
// (e.g., connection, RPC). It can be closed and sent byte representations.
type Peer interface {
	io.Closer
	Send(context.Context, []byte) error
}

const defaultWriteTimeout = 5 * time.Second

type TCPPeer struct {
	net.Conn
	RemoteAddress net.Addr
}

func NewTCPPeer(conn net.Conn) (*TCPPeer, error) {
	// if network := conn.RemoteAddr().Network(); network != "tcp" && network != "tcp4" && network != "tcp6" {
	// 	return nil, fmt.Errorf("connection is not TCP: %s", network)
	// }

	return &TCPPeer{
		Conn:          conn,
		RemoteAddress: conn.RemoteAddr(),
	}, nil

}

func (t *TCPPeer) Send(ctx context.Context, bytes []byte) error {
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
			fmt.Printf("AAAAAA:: %s", err)
			return err
		}
		totalSent += n
	}

	return nil
}

func (t *TCPPeer) Close() error {
	return t.Conn.Close()
}
