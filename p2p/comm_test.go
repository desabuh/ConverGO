package p2p

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTCPServerListen(t *testing.T) {

	listenerAddr := ":9091"

	addr, err := net.ResolveTCPAddr("tcp", listenerAddr)
	assert.Nil(t, err, "listener address should resolve successfully")
	assert.NotNil(t, addr, "resolved address should not be nil")

	transport := NewTCPTransport[struct{}]()

	go transport.ListenFor(addr)

	time.Sleep(300 * time.Millisecond)

	assert.Equal(t, listenerAddr, transport.Address, "server listening interface address match")

}

func TestTCPServerShutdown(t *testing.T) {
	listenerAddr := ":9092"

	addr, err := net.ResolveTCPAddr("tcp", listenerAddr)
	assert.Nil(t, err, "listener address should resolve successfully")
	assert.NotNil(t, addr, "resolved address should not be nil")

	var server = NewTCPTransport[struct{}]()

	terminationChannel := make(chan struct{})

	go func() {
		err := server.ListenFor(addr)

		assert.Nil(t, err, "server was shut down with success")

		close(terminationChannel)

	}()

	time.Sleep(500 * time.Millisecond) // small delay to be sure server is waiting for termination

	server.Shutdown()

	<-terminationChannel

}

func TestTCPReadMsgFromConnection(t *testing.T) {

	listenerAddr := ":9093"

	addr, err := net.ResolveTCPAddr("tcp", listenerAddr)
	assert.Nil(t, err, "listener address should resolve successfully")
	assert.NotNil(t, addr, "resolved address should not be nil")

	event := NewSingleMessageEvent("Hi!", "FILE", "SEND_FILE",
		map[string]interface{}{
			"username": "alice",
			"age":      "30",
		},
	)

	encDec := NewMatchingEncDec(JsonEncoder[Event]{}, JsonDecoder[Event]{})

	var client = NewTCPTransport[Event]()
	var server = NewTCPTransport[Event]()

	defer server.Shutdown()

	client.SetEncoderDecoder(encDec)
	server.SetEncoderDecoder(encDec)

	go server.ListenFor(addr)

	time.Sleep(1 * time.Second)

	go func() {
		defer client.Shutdown()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		err := client.Add(ctx, addr)

		assert.Nil(t, err, "connection with "+listenerAddr+" should succeed")

		err = client.Send(ctx, event, addr)

		assert.Nil(t, err, "Send message should be successfull")

	}()

	select {
	case recEvent := <-server.GetReceiveCh():
		assert.Equal(t, recEvent, event, "received bytestream string should be match")
	case <-time.After(4 * time.Second):
		t.Fatal("timeout waiting for event")
	}

}

func TestTransportWithHandshake(t *testing.T) {
	encDec := NewMatchingEncDec(JsonEncoder[PeerHostInfo]{}, JsonDecoder[PeerHostInfo]{})

	localHostInfoA := PeerHostInfo{
		Id:      "1",
		Name:    "A",
		Address: "localhost:8081",
	}

	addr1, err := net.ResolveTCPAddr("tcp", localHostInfoA.Address)

	require.Nil(t, err, "Address A should be valid not "+addr1.String())

	localHostInfoB := PeerHostInfo{
		Id:      "2",
		Name:    "B",
		Address: "localhost:8082",
	}

	addr2, err := net.ResolveTCPAddr("tcp", localHostInfoB.Address)

	require.Nil(t, err, "Address B should be valid not "+addr2.String())

	localHostInfoC := PeerHostInfo{
		Id:      "3",
		Name:    "C",
		Address: "localhost:8083",
	}

	addr3, err := net.ResolveTCPAddr("tcp", localHostInfoC.Address)

	require.Nil(t, err, "Address C should be valid not "+addr3.String())

	var hostA = NewTCPTransportWithShake[Event](GetNewHostExhanger(localHostInfoA, encDec))
	var hostB = NewTCPTransportWithShake[Event](GetNewHostExhanger(localHostInfoB, encDec))
	var hostC = NewTCPTransportWithShake[Event](GetNewHostExhanger(localHostInfoC, encDec))

	go hostA.ListenFor(addr1)

	go hostB.ListenFor(addr2)

	go hostC.ListenFor(addr3)

	time.Sleep(1 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err = hostA.Add(ctx, addr2)

	require.Nil(t, err, "Peer B should be successfully added")

	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = hostA.Add(ctx, addr3)

	require.Nil(t, err, "Peer C should be successfully added")

	require.True(t, len(hostA.peers) == 2)
	require.True(t, len(hostB.peers) == 1)
	require.True(t, len(hostC.peers) == 1)

	hostA.removePeer(addr2)

	time.Sleep(1 * time.Second)

	require.True(t, len(hostA.peers) == 1)
	require.True(t, len(hostB.peers) == 0)
	require.True(t, len(hostC.peers) == 1)

}
