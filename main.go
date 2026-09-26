package main

import (
	"context"
	"os"

	"github.com/desabuh/convergo/cluster"
	"github.com/desabuh/convergo/command"
	"github.com/desabuh/convergo/config"
	"github.com/desabuh/convergo/file"
	"github.com/desabuh/convergo/log"
	"github.com/desabuh/convergo/p2p"
)

func main() {

	commandFile, exists := os.LookupEnv("COMMAND_FILE")

	var stream *os.File
	if exists {
		var err error
		stream, err = file.OpenFile(commandFile, os.O_RDWR)
		defer stream.Close()

		if err != nil {
			panic(err)
		}
	} else {
		stream = os.Stdin
	}

	var logFactory log.GlobalLoggerFactory = log.NewMutexLoggerFactory(config.LoggerDefaultConfigExtractor{})
	var networkModuleFactory p2p.NetworkModuleFactory[p2p.PeerMessage, p2p.PeerHostInfo, p2p.PeerMetadata] = p2p.TCPJsonPeerNetworkModuleFactory{ConfigExtractor: config.NeworkDefaultConfigExtractor{}}

	var appNodeModuleFactory cluster.AppNodeModuleFactory[p2p.PeerMessage, p2p.PeerHostInfo, p2p.PeerMetadata] = cluster.AppNodeModuleFactory[p2p.PeerMessage, p2p.PeerHostInfo, p2p.PeerMetadata]{NetworkModuleFactory: networkModuleFactory, LoggerFactory: logFactory}

	store := cluster.NewClusterNodeStore(appNodeModuleFactory)

	command.NewCommandLoop(store, command.CommandRegistry, stream, logFactory).
		Run(context.Background())

}
