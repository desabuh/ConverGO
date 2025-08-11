package main

import "github.com/desabuh/convergo/p2p"

func main() {
	listenerAddr := ":9091"

	var server = p2p.NewTCPServer(listenerAddr, p2p.JsonDecoder{})

	go server.ListenFor()

	select {}

}
