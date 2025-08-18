package main

import "github.com/desabuh/convergo/p2p"

func main() {

	var _ = p2p.NewTCPTransport[struct{}]()

	select {}

}
