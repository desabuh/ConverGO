package p2p

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"github.com/desabuh/convergo/utils"
)

const INCOMING_HANDSHAKE_TIMEOUT = 10 * time.Second

var DEFAULT_OUT_STREAM = os.Stdout

type Flag interface {
	io.Closer
	WaitTermination()
	IsClosed() bool
}

type ShutDownFlag struct {
	shutdownCh chan (struct{})

	mu         sync.RWMutex
	isShutDown bool
}

func (s *ShutDownFlag) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.IsClosed() {
		return errors.New("flag was already shut down")
	}

	s.isShutDown = true
	close(s.shutdownCh)

	return nil
}

func (s *ShutDownFlag) WaitTermination() {
	<-s.shutdownCh
}

func (s *ShutDownFlag) IsClosed() bool {
	return s.isShutDown
}

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
// - SetCodec: Sets the encoder/decoder for serializing and deserializing data.
// - SetLocalPingFactory: provides a way for the transport to locally construct a message D from an any value R without redirecting an explicit receiving message
type Transport[D any, T comparable, R any] interface {
	ListenFor(id T) error
	Send(ctx context.Context, data D, target T) error
	Broadcast(ctx context.Context, data D) error
	Add(ctx context.Context, target T) error
	Remove(target T) error
	GetReceiveCh() <-chan D
	Shutdown() error
	SetCodec(encDec Codec[D])
	SetLocalPingFactory(msgFun func(input R) D)
}

type TCPLayer[D any] struct {
	listener net.Listener
	Address  string
	shutdown Flag
	msgCh    chan D

	localMsgFactory func(metadata PeerMetadata) D
	isFactorySet    bool

	handshaker HandShaker[net.Conn]

	endDec Codec[D]

	mu    sync.Mutex
	peers map[net.Addr]Peer
}

// create a simple transport layer with no handshake measure
func NewTCPTransport[D any]() *TCPLayer[D] {
	return NewTCPTransportWithShake[D](&NopHandshaker{})

}

// create a transport layer with a provided handshaker
func NewTCPTransportWithShake[D any](handshaker HandShaker[net.Conn]) *TCPLayer[D] {
	return &TCPLayer[D]{
		shutdown: &ShutDownFlag{
			shutdownCh: make(chan struct{}),
		},
		handshaker: handshaker,
		peers:      make(map[net.Addr]Peer),
		msgCh:      make(chan D),
	}
}

// create a transport layer with an host exhange handshaker
func NewTCPTransportWithHostExhange[D any](info PeerHostInfo, codec Codec[PeerHostInfo]) *TCPLayer[D] {
	return NewTCPTransportWithShake[D](GetNewHostExhanger(info, codec))

}

func (t *TCPLayer[D]) SetCodec(encDec Codec[D]) {
	t.endDec = encDec
}

func (t *TCPLayer[D]) SetLocalPingFactory(factory func(metadata PeerMetadata) D) {
	t.localMsgFactory = factory
	t.isFactorySet = true
}

func (t *TCPLayer[D]) Send(ctx context.Context, data D, target net.Addr) error {
	byteMsg, err := t.endDec.Encode(data)

	if err != nil {
		return err
	}

	if !t.isPeerPresent(target) {
		return fmt.Errorf("target peer %v already exists", target)
	}

	peer := t.peers[target]

	err = peer.Send(ctx, byteMsg)

	if err != nil {
		return err
	}

	return nil
}

func (t *TCPLayer[D]) Broadcast(ctx context.Context, data D) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.peers) == 0 {
		return fmt.Errorf("canno broadcast, no peers are registered")
	}

	for addr := range t.peers {
		err := t.Send(ctx, data, addr)

		if err != nil {
			return err
		}
	}

	return nil
}

func (t *TCPLayer[D]) ListenFor(id net.Addr) error {
	var err error

	t.Address = id.String()

	t.listener, err = net.Listen("tcp", t.Address)

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

			//this context should not be cancelled, it's only intended use is to define timeout con incoming handshaking requests
			ctx, cancel := context.WithTimeout(context.Background(), INCOMING_HANDSHAKE_TIMEOUT)
			defer cancel()
			peer, err := t.handshaker.Shake(ctx, conn, Receiver)

			//var data D = MsgMan.getMessage("PUSH", isSuccess = True, args)
			// exit, broadcast
			//t.Send()
			//t.Send(ctx, nil, peer.GetPeerInfo().Address)

			if err != nil {
				fmt.Printf("Incoming peer connection request failed: %v", err)
				continue
			}

			utils.Logger.NlLog(DEFAULT_OUT_STREAM, "Completed incoming shake with %s", peer.GetPeerInfo().Format())

			t.addPeer(peer.GetPeerInfo().Address, peer)

			ctx, cancel = context.WithCancel(context.Background())
			go func(p Peer) {
				defer cancel()
				t.handlePeer(ctx, p)
			}(peer)
		}
	}()

	t.shutdown.WaitTermination()

	close(t.msgCh)

	return nil

}

func (t *TCPLayer[D]) handlePeer(ctx context.Context, p Peer) {
	defer t.removePeer(p.GetPeerInfo().Address)
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

			if err != nil {
				utils.Logger.NlLog(DEFAULT_OUT_STREAM, "Error received from peer %s: %v", p.GetPeerInfo().Format(), err)
			}

			if !ok {
				utils.Logger.NlLog(DEFAULT_OUT_STREAM, "Connection forcibly closed from %s", p.GetPeerInfo().Format())
				return
			}

		case <-ctx.Done():
			return
		}
	}

}

func (t *TCPLayer[D]) connectTo(ctx context.Context, address string) (Peer, error) {
	conn, err := net.Dial("tcp", address)

	if err != nil {
		return nil, err
	}

	peer, err := t.handshaker.Shake(ctx, conn, Initiator)

	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("Connection closed: %w", err)
	}

	utils.Logger.NlLog(DEFAULT_OUT_STREAM, "Completed outgoing shake with %s", peer.GetPeerInfo().Format())

	return peer, nil
}

func (t *TCPLayer[D]) Add(ctx context.Context, target net.Addr) error {

	if target.String() == t.Address {
		return fmt.Errorf("Cannot connect to %s (same address as local listening address)", t.Address)
	}

	if t.isPeerPresent(target) {
		return fmt.Errorf("target peer %v already exists", target)
	}

	peer, err := t.connectTo(ctx, target.String())

	if err != nil {
		return err
	}

	t.addPeer(target, peer)

	cttx, can := context.WithCancel(context.Background())
	go func(p Peer) {
		defer can()
		t.handlePeer(cttx, p)
	}(peer)

	if t.isFactorySet {
		t.msgCh <- t.localMsgFactory(GetPeerMetadata("PING_HANDSHAKE_SUCCESS", "", peer.GetPeerInfo()))
	}

	return nil
}

func (t *TCPLayer[D]) Remove(target net.Addr) error {

	if !t.isPeerPresent(target) {
		return fmt.Errorf("target peer %v does not exist", target)
	}

	t.removePeer(target)
	return nil
}

func (t *TCPLayer[D]) GetReceiveCh() <-chan D {
	return t.msgCh
}

func (t *TCPLayer[D]) Shutdown() error {
	return t.shutdown.Close()
}

func (t *TCPLayer[D]) isPeerPresent(target net.Addr) bool {
	_, ok := t.peers[target]
	return ok
}

func (t *TCPLayer[D]) addPeer(addr net.Addr, p Peer) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.peers[addr] = p
}

func (t *TCPLayer[D]) removePeer(addr net.Addr) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if peer, ok := t.peers[addr]; ok {

		utils.Logger.NlLog(DEFAULT_OUT_STREAM, "Closing connection with %s peer...", peer.GetPeerInfo().Format())
		peer.Close()

		delete(t.peers, addr)
	}

}
