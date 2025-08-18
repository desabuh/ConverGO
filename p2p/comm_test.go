package p2p

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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

		err := client.Add(addr)

		assert.Nil(t, err, "connection with "+listenerAddr+" should succeed")

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		err = client.Send(ctx, event, addr)

		assert.Nil(t, err, "Send message should be successfull")

	}()

	for recEvent := range server.GetReceiveCh() {
		assert.Equal(t, recEvent, event, "received bytestream string should be match")

		break
	}

}
