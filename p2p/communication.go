package p2p

import (
	"context"
	"fmt"
	"net"
	"sync"
)

// Transport defines an interface for a communication mechanism that can send and receive data
// between different targets. It is generic over the data type D and the target type T.
//
// D represents the type of data being transmitted.
// T represents the type used to identify communication targets and the local reference.
//
// Methods:
// - ListenFor: Starts listening for incoming data or connections by providing an id reference.
// - Send: Sends the specified data to the given target.
// - Add: Adds a target to the list of communication targets.
// - Remove: Removes a target from the list of communication targets.
// - GetReceiveCh: Returns a channel for receiving incoming data.
// - Shutdown: Gracefully shuts down the transport mechanism.
// - SetEncoderDecoder: Sets the encoder/decoder for serializing and deserializing data.
type Transport[D any, T comparable] interface {
	ListenFor(id T) error
	Send(ctx context.Context, data D, target T) error
	Add(target T) error
	Remove(target T) error
	GetReceiveCh() <-chan D
	Shutdown() error
	SetEncoderDecoder(encDec EncoderDecoder[D])
}

type TCPEventLayer struct {
	listener net.Listener
	Address  string
	shutdown Flag
	msgCh    chan Event

	endDec EncoderDecoder[Event]

	mu    sync.Mutex
	peers map[net.Addr]Peer
}

func NewTCPEventTransport() *TCPEventLayer {
	return &TCPEventLayer{
		shutdown: &ShutDownFlag{
			shutdownCh: make(chan struct{}),
		},
		peers: make(map[net.Addr]Peer),
		msgCh: make(chan Event),
	}
}

func (t *TCPEventLayer) SetEncoderDecoder(encDec EncoderDecoder[Event]) {
	t.endDec = encDec
}

func (t *TCPEventLayer) Send(ctx context.Context, event Event, target net.Addr) error {
	byteMsg, err := t.endDec.Encode(event)

	if err != nil {
		return err
	}

	if t.isPeerPresent(target) {
		return fmt.Errorf("target peer %v already exists", target)
	}

	peer := t.peers[target]

	err = peer.Send(ctx, byteMsg)

	if err != nil {
		return err
	}

	return nil
}

func (t *TCPEventLayer) ListenFor(id net.Addr) error {
	var err error
	t.listener, err = net.Listen("tcp", id.String())

	if err != nil {
		return fmt.Errorf("listening error: %s", err)

	}

	defer t.listener.Close()

	go func() {
		for {
			conn, err := t.listener.Accept()

			if t.shutdown.IsClosed() {
				break
			}

			if err != nil {
				fmt.Println("connection accept error:", err)
				continue
			}

			peer, err := NewTCPPeer(conn)
			if err != nil {
				fmt.Printf("Cannot create a new TCP peer: %s", err)
			}

			t.addPeer(peer.RemoteAddress, peer)

			ctx, cancel := context.WithCancel(context.Background())
			go func(p Peer) {
				defer cancel()
				t.handlePeer(ctx, peer)
			}(peer)
		}
	}()

	t.shutdown.WaitTermination()

	return nil

}

func (t *TCPEventLayer) handlePeer(ctx context.Context, p Peer) {
	defer t.removePeer(p.(*TCPPeer).RemoteAddress)
	defer p.Close()

	peerMsgCh, peerErrCh := p.Receive(ctx)

	for {

		if t.shutdown.IsClosed() {
			return
		}

		select {
		case msg, ok := <-peerMsgCh:
			if !ok {
				return
			}
			data, err := t.endDec.Decode(msg)

			if err == nil {
				t.msgCh <- data
			}

		case err, ok := <-peerErrCh:
			fmt.Printf("peer %v error: %v\n", p, err)
			if !ok {
				return
			}

		case <-ctx.Done():
			return
		}
	}

}

func (t *TCPEventLayer) connectTo(address string) (Peer, error) {
	conn, err := net.Dial("tcp", address)

	if err != nil {
		return nil, err
	}

	peer, err := NewTCPPeer(conn)
	if err != nil {
		fmt.Printf("Cannot create a new TCP peer, connection aborted: %s", err)
		conn.Close()
	}

	return peer, nil
}

func (t *TCPEventLayer) Add(target net.Addr) error {
	if t.isPeerPresent(target) {
		return fmt.Errorf("target peer %v already exists", target)
	}

	peer, err := t.connectTo(target.String())

	if err != nil {
		return err
	}

	t.addPeer(target, peer)

	return nil
}

func (t *TCPEventLayer) Remove(target net.Addr) error {

	if !t.isPeerPresent(target) {
		return fmt.Errorf("target peer %v does not exist", target)
	}

	t.removePeer(target)
	return nil
}

func (t *TCPEventLayer) GetReceiveCh() <-chan Event {
	return t.msgCh
}

func (t *TCPEventLayer) Shutdown() error {
	return t.shutdown.Close()
}

func (t *TCPEventLayer) isPeerPresent(target net.Addr) bool {
	_, ok := t.peers[target]
	return ok
}

func (t *TCPEventLayer) addPeer(addr net.Addr, p Peer) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.peers[addr] = p
}

func (t *TCPEventLayer) removePeer(addr net.Addr) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.peers, addr)
}

func main() {
	var t Transport[Event, net.Addr] = NewTCPEventTransport()
	addr := &net.TCPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 8080,
	}
	err := t.ListenFor(addr)
	if err != nil {
		fmt.Printf("Error listening on address %v: %v\n", addr, err)
		return
	}

	fmt.Print(t)
}
