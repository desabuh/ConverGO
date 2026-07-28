package p2p

import (
	"context"
	"io"
	"net"

	"github.com/desabuh/convergo/utils"
)

// Simple interface to define an handshake procedure: it accept a context, a io.Closer and a role and return a peer (remote connection) and an error.
type HandShaker[T io.Closer] interface {
	Shake(ctx context.Context, conn T, role HandshakeRole) (Peer, error)
}

type HandshakeRole int

const (
	Initiator HandshakeRole = iota
	Receiver
)

type HostExhangerStep string

const (
	StepInit            HostExhangerStep = "init"
	StepReceiverExhange HostExhangerStep = "receiver exhange"
	StepSenderExhange   HostExhangerStep = "sender exhange"
	StepACK             HostExhangerStep = "ack"
)

func NewHandshakeError(step HostExhangerStep, address string, err error) error {
	if err == nil {
		return nil
	}
	return &utils.HandshakeError{
		Step:   string(step),
		Target: address,
		Err:    err,
	}
}

// mocked Handshaker that does nothing, apart from wrap a connection over a peer
type NopHandshaker struct{}

func (ex *NopHandshaker) Shake(ctx context.Context, conn net.Conn, role HandshakeRole) (Peer, error) {
	peer, err := NewTCPPeer(conn)

	if err != nil {
		return nil, NewHandshakeError(StepInit, "unknown", err)
	}

	if ctx.Err() != nil {
		return nil, NewHandshakeError(StepInit, "unknown", ctx.Err())
	}

	return peer, nil
}
