package p2p

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestTCPServerListen(t *testing.T) {

	listenerAddr := ":9091"

	var server = NewTCPServer(listenerAddr)

	go server.ListenFor()

	assert.Equal(t, listenerAddr, server.Address, "server listening interface address match")

}

func TestTCPServerShutdown(t *testing.T) {
	listenerAddr := ":9092"

	var server = NewTCPServer(listenerAddr)

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
	msgStr := "Test Message"

	msgCh := make(chan []byte)

	var server = NewTCPServerWithMsgCh(listenerAddr, msgCh)

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

		byteMsg := []byte(msgStr)
		n, err := conn.Write(byteMsg)

		assert.Nil(t, err, "bytestrem ("+msgStr+") should have been written with success")
		assert.Equal(t, len(byteMsg), n, "number of bytes written should match the length of msgStr")

	}()

	for msg := range msgCh {
		assert.Equal(t, string(msg), msgStr, "received bytestream string should be "+msgStr)
		break
	}

}

func TestEncodeDecodeJsonMsg(t *testing.T) {
	listenerAddr := ":9094"

	msg := Message{
		PayLoad:   "Test Message",
		Protocoll: "default",
		Sender: UserData{
			Id:       uuid.New(),
			Username: "TestUser",
		},
	}

	encoder := JsonEncoder{}
	decoder := JsonDecoder{}

	msgCh := make(chan []byte)

	var server = NewTCPServerWithMsgCh(listenerAddr, msgCh)

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

	for msgFromByte := range msgCh {

		msgFrom, err := decoder.Decode(msgFromByte)

		assert.Nil(t, err, "Message should have been decoded with success")

		assert.Equal(t, msg, msgFrom, "Message received should match message sent")
		break
	}
}
