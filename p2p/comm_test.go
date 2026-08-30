package p2p

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/desabuh/convergo/config"
	"github.com/desabuh/convergo/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// a mocked communication pipe thought for failing automatically after limit consecutive send
type MockLimitedPipe[D any, T comparable] struct {
	limit   int
	targets []T
	msgCh   <-chan D
}

func (mp *MockLimitedPipe[D, T]) Send(ctx context.Context, data D, target T) error {
	return nil
}

func (mp *MockLimitedPipe[D, T]) Broadcast(ctx context.Context, data D) error {
	if len(mp.targets) == 0 {
		return fmt.Errorf("cannot broadcast, no targets available")
	}

	for i, target := range mp.targets {
		if i >= mp.limit {
			return &utils.BroadcastError[T]{
				Targets:     mp.targets,
				FailedIndex: i,
				Cause:       fmt.Errorf("mock send limit reached (%d)", mp.limit),
			}
		}

		if err := mp.Send(ctx, data, target); err != nil {
			return &utils.BroadcastError[T]{
				Targets:     mp.targets,
				FailedIndex: i,
				Cause:       err,
			}
		}
	}

	return nil
}

func (t *MockLimitedPipe[D, T]) GetReceiveCh() <-chan D {
	return nil
}

// basic configuration
//var MockTransportConfig config.TransportConfig = config.TransportDefaultConfigExtractor{}.ExtractFrom(map[string]any{})

func getTestHostInfo(address string) PeerHostInfo {
	const TEST_ID = "1"
	const TEST_NAME = "A"

	return CreateNewHostInfo(TEST_ID, TEST_NAME, address)

}

func getTestHostInfo2(address string) PeerHostInfo {
	const TEST_ID = "2"
	const TEST_NAME = "B"

	return CreateNewHostInfo(TEST_ID, TEST_NAME, address)

}

func getTestHostInfo3(address string) PeerHostInfo {
	const TEST_ID = "3"
	const TEST_NAME = "C"

	return CreateNewHostInfo(TEST_ID, TEST_NAME, address)

}

func TestTCPServerListen(t *testing.T) {

	listenerAddr := ":9091"

	transport := NewTCPTransport[struct{}]()

	go transport.ListenFor(getTestHostInfo(listenerAddr))

	time.Sleep(300 * time.Millisecond)

	assert.Equal(t, listenerAddr, transport.LocalID.Address, "server listening interface address match")

}

func TestTCPServerShutdown(t *testing.T) {
	listenerAddr := ":9092"

	var server = NewTCPTransport[struct{}]()

	terminationChannel := make(chan struct{})

	go func() {
		err := server.ListenFor(getTestHostInfo(listenerAddr))

		assert.Nil(t, err, "server was shut down with success")

		close(terminationChannel)

	}()

	time.Sleep(500 * time.Millisecond) // small delay to be sure server is waiting for termination

	server.Shutdown()

	<-terminationChannel

}

func TestTCPReadMsgFromConnection(t *testing.T) {
	listenerAddr := ":9093"
	msg := "TEST"

	encDec := NewCodec(JsonEncoder[string]{}, JsonDecoder[string]{})

	var client = NewTCPTransport[string]()
	var server = NewTCPTransport[string]()

	defer server.Shutdown()
	defer client.Shutdown()

	client.SetCodec(encDec)
	server.SetCodec(encDec)

	info := getTestHostInfo(listenerAddr)

	go server.ListenFor(info)

	time.Sleep(1 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.Add(ctx, info)

	assert.Nil(t, err, "connection with "+listenerAddr+" should succeed")

	err = client.Send(ctx, msg, info)

	assert.Nil(t, err, "Send message should be successfull")

	select {
	case recMsg := <-server.GetReceiveCh():
		assert.Equal(t, recMsg, msg, "received bytestream string should match")
	case <-ctx.Done():
		t.Fatal("timeout waiting for message")
	}

}

func TestTransportWithHandshake(t *testing.T) {

	addressA := "localhost:8081"

	addressB := "localhost:8082"

	addressC := "localhost:8083"

	localhostInfoA := getTestHostInfo(addressA)

	localhostInfoB := getTestHostInfo2(addressB)

	localhostInfoC := getTestHostInfo3(addressC)

	transportCodec := NewCodec(JsonEnvelopeEncoder[PeerMetadata]{}, JsonEnvelopeDecoder[PeerMetadata]{})

	var hostA = NewTCPTransportWithHostExhange(localhostInfoA, transportCodec)
	var hostB = NewTCPTransportWithHostExhange(localhostInfoB, transportCodec)
	var hostC = NewTCPTransportWithHostExhange(localhostInfoC, transportCodec)

	go hostA.ListenFor(localhostInfoA)

	go hostB.ListenFor(localhostInfoB)

	go hostC.ListenFor(localhostInfoC)

	time.Sleep(1 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := hostA.Add(ctx, localhostInfoB)

	require.Nil(t, err, "Peer B should be successfully added")

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = hostA.Add(ctx, localhostInfoC)

	require.Nil(t, err, "Peer C should be successfully added")

	require.True(t, len(hostA.peers) == 2)
	require.True(t, len(hostB.peers) == 1)
	require.True(t, len(hostC.peers) == 1)

	hostA.removePeer(localhostInfoB)

	time.Sleep(1 * time.Second)

	require.True(t, len(hostA.peers) == 1)
	require.True(t, len(hostB.peers) == 0)
	require.True(t, len(hostC.peers) == 1)

}

func TestBroadcastError(t *testing.T) {

	const LIMIT_TO_FAIL = 2

	const TAR1 string = "target-1"
	const TAR2 = "target-2"
	const TAR3 = "target-3"
	const TAR4 = "target-4"

	const MOCK_MSG = ""

	var pipe CommunicationPipe[string, string] = &MockLimitedPipe[string, string]{
		limit:   LIMIT_TO_FAIL,
		targets: []string{TAR1, TAR2, TAR3, TAR4},
	}

	err := pipe.Broadcast(context.Background(), MOCK_MSG)

	require.NotNil(t, err)

	var be *utils.BroadcastError[string]
	if !errors.As(err, &be) {
		t.Fatalf("error returned should be a BroadcastError")
	}

	require.Equal(t, TAR3, be.FailedTarget()) //broadcast should fail at LIMIT_TO_FAIL + 1

	require.Equal(t, []string{TAR1, TAR2}, be.SuccessTargets()) //first LIMIT_TO_FAIL targets should be send without problem

	require.Equal(t, []string{TAR3, TAR4}, be.NotSent()) // failed targets onwards should be dropped

}

func TestMultipleSessionMessages(t *testing.T) {

	var networkModuleFactory NetworkModuleFactory[PeerMessage, PeerHostInfo, PeerMetadata] = TCPJsonPeerNetworkModuleFactory{ConfigExtractor: config.NeworkDefaultConfigExtractor{}}

	infoA := getTestHostInfo("localhost:8084")
	infoB := getTestHostInfo("localhost:8085")

	net1 := networkModuleFactory.Create(infoA)
	defer net1.Transport.Shutdown()
	defer net1.MessageBroker.Stop()

	net2 := networkModuleFactory.Create(infoB)
	defer net2.Transport.Shutdown()
	defer net2.MessageBroker.Stop()

	go net1.Transport.ListenFor(infoA)

	go net2.Transport.ListenFor(infoB)

	var topic TopicName = "TEST_TOPIC"

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := net1.Add(ctx, infoB)

	if err != nil {
		t.Fatalf("Cannot add host: %v", err)
	}

	go net1.StartWaiting(ctx)

	go net2.RegisterHandler(
		topic,
		func(ctx context.Context, session Session[PeerMessage, PeerHostInfo], msg PeerMessage) {

			metadata := PeerMetadata{
				PeerHostInfo: infoB,
			}

			err := session.Send(ctx, GetEmptyEnvelope(metadata), msg.Key.PeerHostInfo)

			if err != nil {
				cancel()
			}

		}).StartWaiting(ctx)

	session := net1.CreateNewSession(topic)
	defer session.Close()

	metadata := PeerMetadata{
		PeerHostInfo: infoA,
	}

	err = session.Send(ctx, GetEmptyEnvelope(metadata), infoB)

	if err != nil {
		t.Fatalf("Session send failed: %v", err)
	}

	ctxT, cancelT := context.WithTimeout(ctx, 3*time.Second)
	defer cancelT()

	env, err := session.WaitOn(ctxT)

	if err != nil {
		t.Fatalf("Session retrieve msg failed: %v", err)
	}

	assert.Equal(t, infoB, env.Key.PeerHostInfo)

}
