package command

import (
	"strconv"
	"time"

	"github.com/desabuh/convergo/cluster"
	"github.com/desabuh/convergo/config"
	"github.com/desabuh/convergo/history"
	"github.com/desabuh/convergo/log"
	"github.com/desabuh/convergo/p2p"
)

var logFactory log.GlobalLoggerFactory = log.NewMutexLoggerFactory(config.LoggerDefaultConfigExtractor{})
var networkModuleFactory p2p.NetworkModuleFactory[p2p.PeerMessage, p2p.PeerHostInfo, p2p.PeerMetadata] = p2p.TCPJsonPeerNetworkModuleFactory{ConfigExtractor: config.NeworkDefaultConfigExtractor{}}

var appNodeModuleFactory cluster.AppNodeModuleFactory[p2p.PeerMessage, p2p.PeerHostInfo, p2p.PeerMetadata] = cluster.AppNodeModuleFactory[p2p.PeerMessage, p2p.PeerHostInfo, p2p.PeerMetadata]{NetworkModuleFactory: networkModuleFactory, LoggerFactory: logFactory}

//var clusterClient *cluster.ClusterClient

var CommandRegistry = NewCommandParser[*cluster.ClusterNodeStore]().
	Register(
		"edit",
		func(cmd CommandContext[*cluster.ClusterNodeStore]) {
			const NUM_PARAMS = 3
			const INSERT_NUM_PARAMS = 4

			if len(cmd.Args) < NUM_PARAMS {
				cmd.Reply(FromErrStr("Command should have at least %d arguments", NUM_PARAMS))
				return
			}

			opType := cmd.Args[0]

			filepath := cmd.Args[1]

			pos, err := strconv.Atoi(cmd.Args[2])

			if err != nil {
				cmd.Reply(FromErrStr("Position should be an integer"))
				return
			}

			content := ""
			if opType == "insert" {
				if len(cmd.Args) < INSERT_NUM_PARAMS {
					cmd.Reply(FromErrStr("Command edit should also provide 'content' parameter with insertion"))
					return
				}
				content = cmd.Args[3]
			}

			client, err := cmd.App.GetDefaultClient()

			if err != nil {
				cmd.Reply(FromErrStr("No working client was set: %v", err))
			}

			wasFileCreated, err := client.CreateNewLocalOperation(filepath, opType, content, pos)

			if err != nil {

				var errorLog ResultMessage = FromErrStr("Fail to edit operation %s for file %s", opType, filepath)

				if wasFileCreated {
					cmd.Reply(ConcatResultMessages(errorLog, FromErrStr("An error occured when trying to create file %s: %v", filepath, err)))
				} else {
					cmd.Reply(ConcatResultMessages(errorLog, FromErrStr("An error occured when trying to update file %s: %v", filepath, err)))
				}

				return

			}

			cmd.Reply(FromSuccess("Edit operation for %s file context was a success", nil, filepath))

		}).
	Register(
		"push",
		func(cmd CommandContext[*cluster.ClusterNodeStore]) {

			client, err := cmd.App.GetDefaultClient()

			if err != nil {
				cmd.Reply(FromErrStr("No working client was set: %v", err))
			}

			err = client.DiffuseData(cmd.Context)

			if err != nil {
				cmd.Reply(FromErrStr("Push broadcasting failure: %v", err))
				return
			}

			cmd.Reply(FromSuccess("Push broadcasting successfull!", nil))

		}).
	Register(
		"read",
		func(cmd CommandContext[*cluster.ClusterNodeStore]) {

			if len(cmd.Args) < 1 {
				cmd.Reply(FromErrStr("File context <filepath> argument should be supplied"))
				return
			}

			filepath := cmd.Args[0]

			client, err := cmd.App.GetDefaultClient()

			if err != nil {
				cmd.Reply(FromErrStr("No working client was set: %v", err))
			}

			content, err := client.DisplayData(history.ContentVisual, filepath, 0)

			if err != nil {
				cmd.Reply(FromErrStr("error when trying to display visual content for %s: %v", filepath, err))
				return
			}

			cmd.Reply(FromSuccess("File %s was read with success", content, filepath))

		}).
	Register(
		"history",
		func(cmd CommandContext[*cluster.ClusterNodeStore]) {

			if len(cmd.Args) < 1 {
				cmd.Reply(FromErrStr("Should specify file context <pathfile>"))
				return
			}

			filepath := cmd.Args[0]

			var deep int = 0
			var mode history.OpVisualizationMode = history.LinearVisual

			if len(cmd.Args) > 1 {

				var err error
				deep, err = strconv.Atoi(cmd.Args[1])

				if err != nil {
					cmd.Reply(FromErrStr("Argument <deep> should be specified as an integer"))
					return
				}

				mode = history.DagVisual
			}

			client, err := cmd.App.GetDefaultClient()

			if err != nil {
				cmd.Reply(FromErrStr("No working client was set: %v", err))
				return
			}

			res, err := client.DisplayData(mode, filepath, deep)

			if err != nil {
				cmd.Reply(FromErrStr("History for file %s could not be correctly visualized: %v", filepath, err))
				return
			}

			cmd.Reply(FromSuccess(
				"Visualizing %s history:",
				res,
				filepath,
			))

		}).
	Register(
		"create_cluster_client",
		func(cmd CommandContext[*cluster.ClusterNodeStore]) {

			if len(cmd.Args) == 0 {
				cmd.Reply(FromErrStr("First argument should specify client info <name>_<address>"))
				return
			}

			clientInfo, err := p2p.ParseHostInfoFromStr(cmd.Args[0], p2p.CLUSTER_CLIENT_ID)

			if err != nil {
				cmd.Reply(FromError(err))
				return
			}

			if len(cmd.Args) < 2 {
				cmd.Reply(FromErrStr("Second argument should specify server info <domain>_<address>"))
				return
			}

			serverInfo, err := p2p.ParseHostInfoFromStr(cmd.Args[1], p2p.CLUSTER_SERVER_ID)

			if err != nil {
				cmd.Reply(FromError(err))
				return
			}

			err = cmd.App.CreateNewClient(cmd.Context, serverInfo, clientInfo)

			if err != nil {
				cmd.Reply(FromErrStr("Cannot create a new Client: %v", err))
				return
			}

			err = cmd.App.SetDefaultClient(clientInfo)

			if err != nil {
				cmd.Reply(FromErrStr("Cannot set working client: %v", err))
				return
			}

			cmd.Reply(FromSuccess("Cluster Client %s was created", nil, clientInfo.Format()))

		}).
	Register(
		"create_cluster_server",
		func(cmd CommandContext[*cluster.ClusterNodeStore]) {

			if len(cmd.Args) == 0 {
				cmd.Reply(FromErrStr("Argument needs to be provided as <domain>_<address>"))
				return
			}

			serverInfo, err := p2p.ParseHostInfoFromStr(cmd.Args[0], p2p.CLUSTER_SERVER_ID)

			if err != nil {
				cmd.Reply(FromError(err))
				return
			}

			err = cmd.App.CreateNewServer(cmd.Context, serverInfo)

			if err != nil {
				cmd.Reply(FromErrStr("Cannot create a new Server: %v", err))
				return
			}

			//server := cluster.CreateNewClusterServer(serverInfo, appNodeModuleFactory)

			//go server.Init(cmd.Context)

			cmd.Reply(FromSuccess("Cluster Server %s was created", nil, serverInfo.Format()))

		}).
	Register(
		"connect_to_cluster",
		func(cmd CommandContext[*cluster.ClusterNodeStore]) {

			if len(cmd.Args) < 1 {
				cmd.Reply(FromErrStr("A new peer address should be provided to connect to cluster"))
				return
			}

			peerAddress := cmd.Args[0]

			//err := clusterClient.ConnectToCluster(cmd.Context, peerAddress)

			client, err := cmd.App.GetDefaultClient()

			if err != nil {
				cmd.Reply(FromErrStr("No working client was set: %v", err))
			}

			err = client.ConnectToCluster(cmd.Context, peerAddress)

			if err != nil {
				cmd.Reply(FromErrStr("Cluster client fail to connect to cluster server: %v", err))
				return
			}

			cmd.Reply(FromSuccess("Connection to cluster server successfull", nil))

		}).
	Register(
		"wait",
		func(cmd CommandContext[*cluster.ClusterNodeStore]) {

			if len(cmd.Args) < 1 {
				cmd.Reply(FromErrStr("A time to wait should be supplied in ms"))
			}

			timeStr := cmd.Args[0]

			timeToWait, err := strconv.Atoi(timeStr)

			if err != nil {
				cmd.Reply(FromErrStr("Time supplied should be an integer: %w", err))
			}

			if timeToWait < 0 {
				cmd.Reply(FromErrStr("Time supplied should be greater than 0"))
			}

			time.Sleep(time.Duration(timeToWait) * time.Millisecond)

			cmd.Reply(FromSuccess("Successfully waited %d ms", nil, timeToWait))

		}).
	Register(
		"set_working_client",
		func(cmd CommandContext[*cluster.ClusterNodeStore]) {

			if len(cmd.Args) == 0 {
				cmd.Reply(FromErrStr("First argument should specify client info as <name>_<address>"))
				return
			}

			clientInfo, err := p2p.ParseHostInfoFromStr(cmd.Args[0], p2p.CLUSTER_CLIENT_ID)

			if err != nil {
				cmd.Reply(FromError(err))
				return
			}

			err = cmd.App.SetDefaultClient(clientInfo)

			if err != nil {
				cmd.Reply(FromErrStr("Cannot set working client: %v", err))
			}

			cmd.Reply(FromSuccess("%s was set as default Client", nil, clientInfo.Format()))
		}).
	Register(
		"set_working_server",
		func(cmd CommandContext[*cluster.ClusterNodeStore]) {

			if len(cmd.Args) < 1 {
				cmd.Reply(FromErrStr("First argument should specify client info as <domain>_<address>"))
				return
			}

			serverInfo, err := p2p.ParseHostInfoFromStr(cmd.Args[0], p2p.CLUSTER_SERVER_ID)

			if err != nil {
				cmd.Reply(FromError(err))
				return
			}

			err = cmd.App.SetDefaultServer(serverInfo)

			if err != nil {
				cmd.Reply(FromErrStr("Cannot set working server: %v", err))
			}

			cmd.Reply(FromSuccess("%s was set as default Server", nil, serverInfo.Format()))
		})
