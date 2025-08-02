package test

import (
	"testing"

	"github.com/desabuh/convergo/p2p"
	"github.com/stretchr/testify/assert"
)

func TestTCPServerListen(t *testing.T) {

	listenerAddr := ":9091"

	var server = p2p.NewTCPServer(listenerAddr)

	go server.ListenFor()

	assert.Equal(t, listenerAddr, server.Address, "server listening interface address match")

}

func TestTCPServerShutdown(t *testing.T) {
	listenerAddr := ":9091"

	var server = p2p.NewTCPServer(listenerAddr)

	terminationChannel := make(chan struct{})

	go func() {
		err := server.ListenFor()

		assert.Nil(t, err, "server was shut down with success")

		close(terminationChannel)

	}()

	server.Shutdown()

	<-terminationChannel

}
