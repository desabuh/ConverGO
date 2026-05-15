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

type TCPHostExchanger struct {
	hostInfoEncDec Codec[PeerHostInfo]
	localInfo      PeerHostInfo
}

func GetNewHostExhanger(localInfo PeerHostInfo, encDec Codec[PeerHostInfo]) TCPHostExchanger {
	return TCPHostExchanger{
		localInfo:      localInfo,
		hostInfoEncDec: encDec,
	}
}

func (ex TCPHostExchanger) Shake(ctx context.Context, conn net.Conn, role HandshakeRole) (Peer, error) {

	peer, err := NewTCPPeer(conn)

	if err != nil {
		return nil, NewHandshakeError(StepInit, peer.RemoteAddress.String(), err)
	}

	remoteAddress := conn.RemoteAddr().String()

	if role == Initiator {
		return ex.initiateHandshake(ctx, peer, remoteAddress)
	}

	return ex.receiveHandshake(ctx, peer, remoteAddress)

}

func (ex TCPHostExchanger) initiateHandshake(ctx context.Context, peer Peer, remoteAddress string) (Peer, error) {

	targetHostname, err := ex.waitForTargetHostName(ctx, peer)

	if err != nil {
		return nil, NewHandshakeError(StepReceiverExhange, remoteAddress, err)
	}

	err = ex.sendHostName(ctx, peer)

	if err != nil {
		return nil, NewHandshakeError(StepSenderExhange, targetHostname.Address.String(), err)
	}

	_, err = ex.waitForTargetHostName(ctx, peer)

	if err != nil {
		return nil, NewHandshakeError(StepACK, targetHostname.Address.String(), err)
	}

	peer.SetPeerInfo(targetHostname)

	return peer, nil
}

func (ex TCPHostExchanger) receiveHandshake(ctx context.Context, peer Peer, remoteAddress string) (Peer, error) {

	err := ex.sendHostName(ctx, peer)

	if err != nil {
		return nil, NewHandshakeError(StepReceiverExhange, remoteAddress, err)
	}

	targetHostname, err := ex.waitForTargetHostName(ctx, peer)

	if err != nil {
		return nil, NewHandshakeError(StepSenderExhange, remoteAddress, err)
	}

	err = ex.sendHostName(ctx, peer) //the hostName is sent another time as ACK

	if err != nil {
		return nil, NewHandshakeError(StepSenderExhange, targetHostname.Address.String(), err)
	}

	peer.SetPeerInfo(targetHostname)

	return peer, nil

}

func (ex *TCPHostExchanger) waitForTargetHostName(ctx context.Context, peer Peer) (PeerHostInfo, error) {
	recCh, errCH := peer.Receive(context.Background()) //TODO

	var byMsg []byte

	select {
	case byMsg = <-recCh:
	case err := <-errCH:
		return PeerHostInfo{}, err
	}

	targetHostName, err := ex.hostInfoEncDec.Decode(byMsg)

	if err != nil {
		return PeerHostInfo{}, err
	}

	return targetHostName, nil
}

func (ex *TCPHostExchanger) sendHostName(ctx context.Context, peer Peer) error {
	byHost, err := ex.hostInfoEncDec.Encode(ex.localInfo)

	if err != nil {
		return err
	}

	err = peer.Send(ctx, byHost)

	if err != nil {
		return err
	}

	return nil

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
