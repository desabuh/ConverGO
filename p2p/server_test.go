package p2p

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTCPServerListen(t *testing.T) {

	listenerAddr := ":9091"

	var server = NewTCPServer(listenerAddr, JsonDecoder{})

	go server.ListenFor()

	assert.Equal(t, listenerAddr, server.Address, "server listening interface address match")

}

func TestTCPServerShutdown(t *testing.T) {
	listenerAddr := ":9092"

	var server = NewTCPServer(listenerAddr, JsonDecoder{})

	terminationChannel := make(chan struct{})

	go func() {
		err := server.ListenFor()

		assert.Nil(t, err, "server was shut down with success")

		close(terminationChannel)

	}()

	time.Sleep(500 * time.Millisecond) // small delay to be sure server is waiting for termination

	server.Shutdown()

	<-terminationChannel

}

func TestTCPReadMsgFromConnection(t *testing.T) {

	listenerAddr := ":9093"

	msgStr := Message{
		Name: "Ping",
		Args: map[string]interface{}{
			"username": "alice",
			"age":      "30",
		},
		Time: 1,
	}

	encoder := JsonEncoder{}

	eventCh := make(chan Event)

	var server = NewTCPServerWithEventCh(listenerAddr, JsonDecoder{}, eventCh)

	go server.ListenFor()

	time.Sleep(1 * time.Second)

	go func() { //TODO: REFACTOR WHEN CLASS CLIENT IS AVAILABLE
		conn, err := net.Dial("tcp", listenerAddr)

		assert.Nil(t, err, "connection with "+listenerAddr+" should succeed")

		defer conn.Close()

		err = conn.SetWriteDeadline(time.Now().Add(3 * time.Second))

		if err != nil {
			fmt.Printf("write deadline met: %v\n", err)
			panic(err)
		}

		byteMsg, err := encoder.Encode(msgStr)

		assert.Nil(t, err, "Message should have been encoded")

		n, err := conn.Write(byteMsg)

		assert.Nil(t, err, "bytestrem should have been written with success")
		assert.Equal(t, len(byteMsg), n, "number of bytes written should match the length of msgStr")

	}()

	for event := range eventCh {
		assert.Equal(t, event.Data, msgStr, "received bytestream string should be match")

		assert.Equal(t, event.Data.Args["username"], "alice", "sender username should match")

		break
	}

}

func TestEncodeDecodeJsonMsg(t *testing.T) {
	listenerAddr := ":9094"

	msg := Message{
		Name: "Ping",
		Args: map[string]interface{}{
			"username": "alice",
			"age":      "30",
		},
		Time: 1,
	}

	encoder := JsonEncoder{}
	decoder := JsonDecoder{}

	eventCh := make(chan Event)

	var server = NewTCPServerWithEventCh(listenerAddr, decoder, eventCh)

	go server.ListenFor()

	time.Sleep(1 * time.Second)

	go func() { //TODO: REFACTOR WHEN CLASS CLIENT IS AVAILABLE
		conn, err := net.Dial("tcp", listenerAddr)

		assert.Nil(t, err, "connection with "+listenerAddr+" should succeed")

		defer conn.Close()

		err = conn.SetWriteDeadline(time.Now().Add(3 * time.Second))

		if err != nil {
			fmt.Printf("write deadline met: %v\n", err)
			panic(err)
		}

		byteMsg, err := encoder.Encode(msg)

		assert.Nil(t, err, "Message should have been encoded with success")

		n, err := conn.Write(byteMsg)

		assert.Nil(t, err, "bytestrem should have been written with success")
		assert.Equal(t, len(byteMsg), n, "number of bytes written should match the length of msgStr")

	}()

	for event := range eventCh {

		assert.Equal(t, msg, event.Data, "Message received should match message sent")
		break
	}
}
