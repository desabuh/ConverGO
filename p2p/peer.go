package p2p

import (
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
}

func NewTCPServer(address string) *TCPServer {
	return &TCPServer{
		Address: address,
		shutdown: &ShutDownFlag{
			shutdownCh: make(chan struct{}),
		},
	}
}

func (t *TCPServer) ListenFor() error {
	var err error
	t.listener, err = net.Listen("tcp", t.Address)

	if err != nil {
		return err

	}

	defer t.listener.Close()

	go func() {
		for {
			conn, err := t.listener.Accept()

			if err != nil {
				fmt.Println("connection accept error:", err)
				continue
			}

			fmt.Print("connection from", conn.RemoteAddr())

			go t.handleConnection(conn, 1024)
		}
	}()

	t.shutdown.WaitTermination()

	return nil

}

func (t *TCPServer) handleConnection(conn net.Conn, buffSize int) {
	defer conn.Close()

	for {
		buf := make([]byte, buffSize)
		n, err := conn.Read(buf)
		if err != nil {
			fmt.Println("byte error read: ", err)
			if t.shutdown.IsClosed() {
				return
			}
		}

		msg := buf[:n]
		fmt.Println(string(msg))
	}

}

func (t *TCPServer) Shutdown() error {
	return t.shutdown.Close()
}
