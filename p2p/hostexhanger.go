package p2p

import (
	"context"
	"fmt"
	"net"
)

const TCP_HOST_EXCHANGER TopicName = "TCP_HOST_EXCHANGER_TOPIC"

type TCPHostExchanger struct {
	encDec    Codec[PeerMessage]
	localInfo PeerHostInfo
}

func GetNewHostExhanger(localInfo PeerHostInfo, encDec Codec[PeerMessage]) TCPHostExchanger {
	return TCPHostExchanger{
		localInfo: localInfo,
		encDec:    encDec,
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

	var currentSeqNo SeqNo = 0

	targetHostname, err := ex.waitForTargetPeerName(ctx, peer, currentSeqNo)

	if err != nil {
		return nil, NewHandshakeError(StepReceiverExhange, remoteAddress, err)
	}

	currentSeqNo++

	err = ex.sendPeerName(ctx, peer, currentSeqNo)

	if err != nil {
		return nil, NewHandshakeError(StepSenderExhange, targetHostname.Address, err)
	}

	currentSeqNo++

	_, err = ex.waitForTargetPeerName(ctx, peer, currentSeqNo)

	if err != nil {
		return nil, NewHandshakeError(StepACK, targetHostname.Address, err)
	}

	peer.SetPeerInfo(targetHostname)

	return peer, nil
}

func (ex TCPHostExchanger) receiveHandshake(ctx context.Context, peer Peer, remoteAddress string) (Peer, error) {

	var currentSeqNo SeqNo = 0

	err := ex.sendPeerName(ctx, peer, currentSeqNo)

	if err != nil {
		return nil, NewHandshakeError(StepReceiverExhange, remoteAddress, err)
	}

	currentSeqNo++

	targetHostname, err := ex.waitForTargetPeerName(ctx, peer, currentSeqNo)

	if err != nil {
		return nil, NewHandshakeError(StepSenderExhange, remoteAddress, err)
	}

	currentSeqNo++

	err = ex.sendPeerName(ctx, peer, currentSeqNo) //the hostName is sent another time as ACK

	if err != nil {
		return nil, NewHandshakeError(StepACK, targetHostname.Address, err)
	}

	peer.SetPeerInfo(targetHostname)

	return peer, nil

}

func (ex *TCPHostExchanger) waitForTargetPeerName(ctx context.Context, peer Peer, expectedSeqNo SeqNo) (PeerHostInfo, error) {
	recCh, errCH := peer.Receive(ctx)

	var byMsg []byte

	select {
	case byMsg = <-recCh:
	case err := <-errCH:
		return PeerHostInfo{}, err
	}

	targetPeerName, err := ex.encDec.Decode(byMsg)

	if err != nil {
		return PeerHostInfo{}, err
	}

	err = ex.checkMsgValidity(expectedSeqNo, targetPeerName.Key.SeqNo, TCP_HOST_EXCHANGER)

	if err != nil {
		return PeerHostInfo{}, err
	}

	return targetPeerName.Key.PeerHostInfo, nil
}

func (ex *TCPHostExchanger) sendPeerName(ctx context.Context, peer Peer, sendingSeqNo SeqNo) error {

	//during handshake only metadata is send but no payload (no additional application level encoding approach is required)
	senderInfo := ex.localInfo
	metadata := GetPeerMetadata(TCP_HOST_EXCHANGER, "", senderInfo)
	metadata.SeqNo = sendingSeqNo

	msg := GetEmptyEnvelope(metadata)

	byPeer, err := ex.encDec.Encode(msg)

	if err != nil {
		return err
	}

	err = peer.Send(ctx, byPeer)

	if err != nil {
		return err
	}

	return nil

}

func (ex *TCPHostExchanger) checkMsgValidity(expectedSeqNo SeqNo, providedSeqNo SeqNo, providedTopicName TopicName) error {
	if providedTopicName != TCP_HOST_EXCHANGER {
		return fmt.Errorf("handshake msg should be send over %s topic not %s", TCP_HOST_EXCHANGER, providedTopicName)
	}

	if providedSeqNo != expectedSeqNo {
		return fmt.Errorf("handshake msg out of sequence (needed %d not %d)", expectedSeqNo, providedSeqNo)
	}

	return nil
}
