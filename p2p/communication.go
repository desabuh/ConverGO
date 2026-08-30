package p2p

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"

	"github.com/desabuh/convergo/config"
	"github.com/desabuh/convergo/log"
	"github.com/desabuh/convergo/utils"
)

var ErrPeerAlreadyExists = errors.New("Peer already Exists")
var ErrPeerDoesNotExists = errors.New("Peer does not exists")

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
// - GetTargets: Return all current targets
// - GetReceiveCh: Returns a channel for receiving incoming data.
// - Shutdown: Gracefully shuts down the transport mechanism.
// - SetCodec: Sets the encoder/decoder for serializing and deserializing data.
// - SetLocalMsgFactory: provides a way for the transport to locally construct a message D from an any value R without redirecting an explicit receiving message
type Transport[D any, T comparable, R any] interface {
	config.Configurable[config.TransportConfig]
	CommunicationPipe[D, T]
	ListenFor(id T) error
	Add(ctx context.Context, target T) error
	Remove(target T) error
	GetTargets() map[T]struct{}
	Shutdown() error
	SetCodec(encDec Codec[D])
	SetLocalMsgFactory(msgFun func(input R, payload any) (D, error))
}

// A generic communication channel where data can be sent or received from
// - Send: Sends the specified data to the given target.
// - Broadcast: Send data to all registered target
// - GetReceiveCh: Returns a channel for receiving incoming data.
type CommunicationPipe[D any, T comparable] interface {
	Send(ctx context.Context, data D, target T) error
	Broadcast(ctx context.Context, data D) error
	GetReceiveCh() <-chan D
}

type TCPLayer[D any] struct {
	listener net.Listener
	LocalID  PeerHostInfo
	shutdown Flag
	msgCh    utils.SafeChannel[D]

	localMsgFactory func(metadata PeerMetadata, payload any) (D, error)
	isFactorySet    bool

	handshaker HandShaker[net.Conn]

	endDec Codec[D]

	config config.TransportConfig

	mu    sync.Mutex
	peers map[PeerHostInfo]Peer
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
		peers:      make(map[PeerHostInfo]Peer),
		config:     config.DefaultTransportConfig,
		msgCh:      utils.NewSafeChannel[D](),
	}
}

// create a transport layer with an host exhange handshaker
func NewTCPTransportWithHostExhange(info PeerHostInfo, codec Codec[PeerMessage]) *TCPLayer[PeerMessage] {
	return NewTCPTransportWithShake[PeerMessage](GetNewHostExhanger(info, codec))
}

func (t *TCPLayer[D]) SetConfig(config config.TransportConfig) {
	t.config = config
}

func (t *TCPLayer[D]) SetCodec(encDec Codec[D]) {
	t.endDec = encDec
}

func (t *TCPLayer[D]) SetLocalMsgFactory(factory func(metadata PeerMetadata, payload any) (D, error)) {
	t.localMsgFactory = factory
	t.isFactorySet = true
}

func (t *TCPLayer[D]) Send(ctx context.Context, data D, target PeerHostInfo) error {
	byteMsg, err := t.endDec.Encode(data)

	if err != nil {
		return err
	}

	if !t.isPeerPresent(target) {
		return ErrPeerDoesNotExists
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

	if len(t.peers) == 0 {
		t.mu.Unlock()
		return fmt.Errorf("cannot broadcast, no peers are registered")
	}

	targets := make([]PeerHostInfo, 0, len(t.peers))
	for addr := range t.peers {
		targets = append(targets, addr)
	}

	t.mu.Unlock()

	for i, addr := range targets {
		err := t.Send(ctx, data, addr)

		if err != nil {
			return &utils.BroadcastError[PeerHostInfo]{
				Targets:     targets,
				FailedIndex: i,
				Cause:       err,
			}
		}
	}

	return nil
}

func (t *TCPLayer[D]) ListenFor(id PeerHostInfo) error {
	var err error

	_, err = net.ResolveTCPAddr("tcp", id.Address)

	if err != nil {
		return fmt.Errorf("provided address %s was not valid", id.Address)
	}

	t.LocalID = id

	t.listener, err = net.Listen("tcp", t.LocalID.Address)

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
			ctx, cancel := context.WithTimeout(context.Background(), t.config.InPairTimeout)
			defer cancel()
			peer, err := t.handshaker.Shake(ctx, conn, Receiver)

			if err != nil {
				fmt.Printf("Incoming peer connection request failed: %v", err)
				conn.Close()
				continue
			}

			log.Logger.NlLog(DEFAULT_OUT_STREAM, "Completed incoming shake with %s", peer.GetPeerInfo().Format())

			t.addPeer(peer.GetPeerInfo(), peer)

			ctx, cancel = context.WithCancel(context.Background())
			go func(p Peer) {
				defer cancel()
				t.handlePeer(ctx, p)
			}(peer)

			t.constructAndSendMetaMsg(GetPeerMetadata("PEER_CONNECTION_TOPIC", "", peer.GetPeerInfo()))
		}
	}()

	t.shutdown.WaitTermination()

	//close(t.msgCh)
	t.msgCh.Close()

	return nil

}

func (t *TCPLayer[D]) handlePeer(ctx context.Context, p Peer) {
	defer t.removePeer(p.GetPeerInfo())

	peerMsgCh, peerErrCh := p.Receive(ctx)

	for {

		if t.shutdown.IsClosed() {
			return
		}

		select {
		case msg, ok := <-peerMsgCh:
			fmt.Printf("TRANSPORT len(msg): %v\n", len(msg))

			if !ok {
				return
			}

			data, err := t.endDec.Decode(msg)

			if err == nil {
				isSendSucc := t.msgCh.Send(data)

				if !isSendSucc {
					return
				}

			}

		case err, ok := <-peerErrCh:

			var loggingErr error

			switch {
			case !ok:
				loggingErr = errors.New("connection forcibly closed")
			case err != nil:
				loggingErr = err
			default:
				loggingErr = errors.New("unknown peer error")
			}

			t.constructAndSendMetaMsg(GetPeerMetadata("PEER_EXIT_TOPIC", loggingErr.Error(), p.GetPeerInfo()))

			return

		case <-ctx.Done():
			return
		}
	}

}

func (t *TCPLayer[D]) constructAndSendMetaMsg(header PeerMetadata) {
	if t.isFactorySet {
		if env, err := t.localMsgFactory(header, nil); err == nil && !t.shutdown.IsClosed() {
			t.msgCh.Send(env)
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

	return peer, nil
}

func (t *TCPLayer[D]) Add(ctx context.Context, target PeerHostInfo) error {

	if target == t.LocalID {
		return fmt.Errorf("Cannot connect to %s (same address as local listening address)", t.LocalID.Format())
	}

	if t.isPeerPresent(target) {
		return ErrPeerAlreadyExists
	}

	ctxT, cancel := context.WithTimeout(ctx, t.config.OutPairtTimeout)
	defer cancel()

	peer, err := t.connectTo(ctxT, target.Address)

	if err != nil {
		return err
	}

	t.addPeer(target, peer)

	// if t.isFactorySet {
	// 	//t.msgCh <- t.localMsgFactory(GetPeerMetadata("PEER_CONNECTION_TOPIC", "", peer.GetPeerInfo()))
	// 	if env, err := t.localMsgFactory(GetPeerMetadata(PEER_CONNECTION, "", peer.GetPeerInfo()), nil); err == nil {
	// 		t.msgCh.Send(env)
	// 	}
	// }
	//t.constructAndSendMetaMsg(GetPeerMetadata(PEER_CONNECTION, "", peer.GetPeerInfo()))

	go func(p Peer) {
		t.handlePeer(ctx, p)
	}(peer)

	return nil
}

func (t *TCPLayer[D]) Remove(target PeerHostInfo) error {

	if !t.isPeerPresent(target) {
		return ErrPeerDoesNotExists
	}

	t.removePeer(target)
	return nil
}

func (t *TCPLayer[D]) GetReceiveCh() <-chan D {
	return t.msgCh.GetReceiveOnlyCh()
}

func (t *TCPLayer[D]) GetTargets() map[PeerHostInfo]struct{} {
	t.mu.Lock()
	defer t.mu.Unlock()

	result := make(map[PeerHostInfo]struct{}, len(t.peers))

	for hostInfo := range t.peers {
		result[hostInfo] = struct{}{}
	}

	return result
}

func (t *TCPLayer[D]) Shutdown() error {

	for addr := range t.peers {
		t.removePeer(addr)
	}

	return t.shutdown.Close()
}

func (t *TCPLayer[D]) isPeerPresent(target PeerHostInfo) bool {
	_, ok := t.peers[target]
	return ok
}

func (t *TCPLayer[D]) addPeer(addr PeerHostInfo, p Peer) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.peers[addr] = p
}

func (t *TCPLayer[D]) removePeer(addr PeerHostInfo) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if peer, ok := t.peers[addr]; ok {
		peer.Close()

		delete(t.peers, addr)
	}

}
