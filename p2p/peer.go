package p2p

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// PeerHostInfo is assumed to be your own type.
// type PeerHostInfo struct { ... }

// Peer represents a remote node view.
type Peer interface {
	io.Closer
	Send(ctx context.Context, data []byte) error

	// Receive starts a logical receive session.
	// At most one session can be active at a time.
	// A new Receive(ctx) will stop the previous session and
	// start a new one with its own channels and context.
	// The session ends when:
	//   - ctx is done (timeout/cancel),
	//   - a new StartReceive is called,
	//   - EOF / connection error,
	//   - Close is called.
	Receive(ctx context.Context) (<-chan []byte, <-chan error)

	SetPeerInfo(info PeerHostInfo)
	GetPeerInfo() PeerHostInfo
}

const (
	defaultWriteTimeout = 5 * time.Second
	MAX_HEADER_BYTE     = 4
	MAX_MESSAGE_SIZE    = 4 * 1024 * 1024
)

// TCPPeer implements Peer over a net.Conn.
type TCPPeer struct {
	mu sync.Mutex
	net.Conn

	remotePeerInfo PeerHostInfo

	streamProcessor ByteStreamProcessor

	// Peer lifetime
	closeOnce sync.Once
	done      chan struct{} // closed when peer is closed

	// Single connection read loop
	readOnce sync.Once

	// Current logical receiver (session)
	recvActive bool
	recvCtx    context.Context
	recvCancel context.CancelFunc
	recvMsgCh  chan []byte
	recvErrCh  chan error

	RemoteAddress net.Addr
}

// NewTCPPeer constructs a TCPPeer from an existing net.Conn.
func NewTCPPeer(conn net.Conn) (*TCPPeer, error) {
	if conn == nil {
		return nil, fmt.Errorf("provided connection is not valid")
	}

	streamProcessor, err := GetNewStreamProcessor(MAX_HEADER_BYTE, MAX_MESSAGE_SIZE)

	if err != nil {
		return nil, err
	}

	return &TCPPeer{
		Conn:            conn,
		RemoteAddress:   conn.RemoteAddr(),
		streamProcessor: streamProcessor,
		done:            make(chan struct{}),
	}, nil
}

// ensureReadLoop starts the single read loop exactly once.
func (t *TCPPeer) ensureReadLoop() {
	t.readOnce.Do(func() {
		go t.readLoop()
	})
}

// Send writes bytes to the underlying connection, honoring ctx for timeout/cancel.
func (t *TCPPeer) Send(ctx context.Context, data []byte) error {
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

	// totalSent := 0
	// for totalSent < len(data) {
	// 	select {
	// 	case <-ctx.Done():
	// 		return ctx.Err()
	// 	default:
	// 	}

	// 	n, err := t.Conn.Write(data[totalSent:])
	// 	if err != nil {
	// 		return err
	// 	}
	// 	totalSent += n
	// }

	return t.streamProcessor.SendByteOnStream(t.Conn, data)

}

// StartReceive starts or replaces a logical receive session.
// If a previous session exists, it is canceled and its channels are closed.
func (t *TCPPeer) Receive(ctx context.Context) (<-chan []byte, <-chan error) {
	t.ensureReadLoop()

	// Create new session context and channels first.
	recvCtx, recvCancel := context.WithCancel(ctx)
	msgCh := make(chan []byte, 16)
	errCh := make(chan error, 1)

	t.mu.Lock()

	// If there is an active session, cancel and close it.
	if t.recvActive {
		if t.recvCancel != nil {
			t.recvCancel()
		}
		t.closeSessionLocked(nil)
	}

	// Install new session.
	t.recvActive = true
	t.recvCtx = recvCtx
	t.recvCancel = recvCancel
	t.recvMsgCh = msgCh
	t.recvErrCh = errCh

	t.mu.Unlock()

	// Watcher: if recvCtx ends while no messages are moving,
	// tear down this session. This ensures ctx timeout alone ends
	// the session even if the connection is idle.
	go func(recvCtx context.Context) {
		<-recvCtx.Done()
		t.mu.Lock()
		if t.recvActive && t.recvCtx == recvCtx {
			t.closeSessionLocked(recvCtx.Err())
		}
		t.mu.Unlock()
	}(recvCtx)

	return msgCh, errCh
}

// readLoop is the single goroutine that reads from net.Conn and
// multiplexes data into the active receive session, if any.
func (t *TCPPeer) readLoop() {
	////buf := make([]byte, 2048)

	for {
		// select {
		// case <-t.done:
		// 	t.mu.Lock()
		// 	t.closeSessionLocked(io.EOF)
		// 	t.mu.Unlock()
		// 	return
		// default:
		// }

		// // n, err := t.Conn.Read(buf)
		// // fmt.Printf("n: %v\n", n)
		// // fmt.Printf("err: %v\n", err)
		// // if err != nil {
		// // 	t.handleReadError(err)
		// // 	return
		// // }

		// // msg := make([]byte, n)
		// // fmt.Printf("len(msg): %v\n", len(msg))

		// // copy(msg, buf[:n])

		msg, err := t.streamProcessor.ReceiveByteFromStream(t.Conn)

		if err != nil {
			t.handleReadError(err)
			return
		}

		t.deliverMessage(msg)
	}
}

// deliverMessage attempts to deliver msg to the active session, if any.
// If there is no active session, msg is dropped.
func (t *TCPPeer) deliverMessage(msg []byte) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.recvActive || t.recvMsgCh == nil {
		// No active receiver: drop the message.
		return
	}

	// If session context is already done, close it and drop the message.
	select {
	case <-t.recvCtx.Done():
		t.closeSessionLocked(t.recvCtx.Err())
		return
	default:
	}

	select {
	case t.recvMsgCh <- msg:
		// Delivered.
	case <-t.recvCtx.Done():
		// Session ended while delivering.
		t.closeSessionLocked(t.recvCtx.Err())
	}
}

// handleReadError is called when Conn.Read returns an error.
func (t *TCPPeer) handleReadError(err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Propagate the error to the active session, if any.
	if t.recvActive && t.recvErrCh != nil {
		select {
		case t.recvErrCh <- err:
		default:
		}
		t.closeSessionLocked(nil)
	}

	// t.closeOnce.Do(func() {
	// 	close(t.done)
	// })
}

// closeSessionLocked closes the current session channels and resets state.
// Caller must hold t.mu.
func (t *TCPPeer) closeSessionLocked(optionalErr error) {
	if !t.recvActive {
		return
	}

	if optionalErr != nil && t.recvErrCh != nil {
		select {
		case t.recvErrCh <- optionalErr:
		default:
		}
	}

	if t.recvMsgCh != nil {
		close(t.recvMsgCh)
	}
	if t.recvErrCh != nil {
		close(t.recvErrCh)
	}

	t.recvActive = false
	t.recvCtx = nil
	t.recvCancel = nil
	t.recvMsgCh = nil
	t.recvErrCh = nil
}

// GetPeerInfo returns the associated PeerHostInfo.
func (t *TCPPeer) GetPeerInfo() PeerHostInfo {
	return t.remotePeerInfo
}

// SetPeerInfo sets the associated PeerHostInfo.
func (t *TCPPeer) SetPeerInfo(info PeerHostInfo) {
	t.remotePeerInfo = info
}

// Close closes the peer and the underlying connection.
func (t *TCPPeer) Close() error {
	t.closeOnce.Do(func() {
		t.mu.Lock()
		t.closeSessionLocked(nil)
		t.mu.Unlock()

		_ = t.Conn.Close()
	})
	return nil
}

// Defines a processor to sending and receiving stream of variable byte size, It creates and read a prefix header for byte length
type ByteStreamProcessor struct {
	headerSizeByte uint32
	maxMessageByte uint32
}

func GetNewStreamProcessor(headerSizeByte uint32, maxMessageByte uint32) (ByteStreamProcessor, error) {
	if headerSizeByte >= maxMessageByte {
		return ByteStreamProcessor{}, fmt.Errorf("header size should be less than max message size")
	}

	return ByteStreamProcessor{headerSizeByte, maxMessageByte}, nil
}

func (b ByteStreamProcessor) SendByteOnStream(conn net.Conn, bytes []byte) error {
	header := make([]byte, b.headerSizeByte)

	binary.BigEndian.PutUint32(header, uint32(len(bytes)))

	if _, err := conn.Write(header); err != nil {
		return err
	}

	_, err := conn.Write(bytes)
	return err

}

func (b ByteStreamProcessor) ReceiveByteFromStream(conn net.Conn) ([]byte, error) {
	header := make([]byte, b.headerSizeByte)

	if _, err := io.ReadFull(conn, header); err != nil {
		return nil, err
	}

	size := binary.BigEndian.Uint32(header)

	if size > b.maxMessageByte {
		return nil, fmt.Errorf("bytestream too large: %d bytes (max is %d)", size, b.maxMessageByte)
	}

	msg := make([]byte, size)

	if _, err := io.ReadFull(conn, msg); err != nil {
		return nil, err
	}

	return msg, nil

}
