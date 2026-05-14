package p2p

import (
	"context"
	"fmt"
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

	encDec := NewCodec(JsonEncoder[Event]{}, JsonDecoder[Event]{})

	var client = NewTCPTransport[Event]()
	var server = NewTCPTransport[Event]()

	defer server.Shutdown()

	client.SetCodec(encDec)
	server.SetCodec(encDec)

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

	idA := "1"
	nameA := "A"
	addressA := "localhost:8081"

	idB := "2"
	nameB := "B"
	addressB := "localhost:8082"

	idC := "3"
	nameC := "C"
	addressC := "localhost:8083"

	localhostInfoA, err := CreateNewHostInfo(idA, nameA, addressA)

	require.Nil(t, err, "Address A should be valid not "+addressA)

	localhostInfoB, err := CreateNewHostInfo(idB, nameB, addressB)

	require.Nil(t, err, "Address B should be valid not "+addressB)

	localhostInfoC, err := CreateNewHostInfo(idC, nameC, addressC)

	require.Nil(t, err, "Address C should be valid not "+addressC)

	var hostA = NewTCPTransportWithHostExhange[PeerHostInfo](localhostInfoA, NewJsonCodec[PeerHostInfo]())
	var hostB = NewTCPTransportWithHostExhange[PeerHostInfo](localhostInfoB, NewJsonCodec[PeerHostInfo]())
	var hostC = NewTCPTransportWithHostExhange[PeerHostInfo](localhostInfoC, NewJsonCodec[PeerHostInfo]())

	go hostA.ListenFor(localhostInfoA.Address)

	go hostB.ListenFor(localhostInfoB.Address)

	go hostC.ListenFor(localhostInfoC.Address)

	time.Sleep(1 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err = hostA.Add(ctx, localhostInfoB.Address)

	require.Nil(t, err, "Peer B should be successfully added")

	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = hostA.Add(ctx, localhostInfoC.Address)

	require.Nil(t, err, "Peer C should be successfully added")

	require.True(t, len(hostA.peers) == 2)
	require.True(t, len(hostB.peers) == 1)
	require.True(t, len(hostC.peers) == 1)

	fmt.Printf("FF localhostInfoB.Address: %v\n", localhostInfoB.Address)
	hostA.removePeer(localhostInfoB.Address)

	time.Sleep(1 * time.Second)

	require.True(t, len(hostA.peers) == 1)
	require.True(t, len(hostB.peers) == 0)
	require.True(t, len(hostC.peers) == 1)

}
