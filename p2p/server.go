package p2p

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
)

type Server interface {
	ListenFor(address string) error
	Shutdown() error
}

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

type TCPServer struct {
	listener net.Listener
	Address  string
	shutdown Flag
	msgCh    chan<- []byte

	mu    sync.Mutex
	peers map[net.Addr]Peer
}

func NewTCPServer(address string) *TCPServer {
	return &TCPServer{
		Address: address,
		shutdown: &ShutDownFlag{
			shutdownCh: make(chan struct{}),
		},
		peers: make(map[net.Addr]Peer),
		msgCh: make(chan<- []byte),
	}
}

func NewTCPServerWithMsgCh(address string, msgCh chan<- []byte) *TCPServer {
	return &TCPServer{
		Address: address,
		shutdown: &ShutDownFlag{
			shutdownCh: make(chan struct{}),
		},
		peers: make(map[net.Addr]Peer),
		msgCh: msgCh,
	}
}

func (t *TCPServer) ListenFor() error {
	var err error
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

func (t *TCPServer) handlePeer(ctx context.Context, p Peer) {
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
			t.msgCh <- msg

		case err, ok := <-peerErrCh:
			if !ok {
				return
			}
			fmt.Printf("peer %v error: %v\n", p, err)

		case <-ctx.Done():
			return
		}
	}

}

func (t *TCPServer) addPeer(addr net.Addr, p Peer) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.peers[addr] = p
}

func (t *TCPServer) removePeer(addr net.Addr) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.peers, addr)
}

func (t *TCPServer) Shutdown() error {
	return t.shutdown.Close()
}
