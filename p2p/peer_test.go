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
