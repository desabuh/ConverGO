package cluster

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/desabuh/convergo/history"
	"github.com/desabuh/convergo/p2p"
)

const (
	CLUSTER_SERVER_RESPONSE_TIME = 6 * time.Second
)

type ClusterInfo struct {
	Domain string
	Id     p2p.PeerHostInfo
}

var (
	CVRDT_PEER_NOT_INITIALIZED = errors.New("Cvrdt peer should be initialized")
)

type ClusterClient struct {
	clusterInfo p2p.PeerHostInfo

	clientActiveFlag atomic.Bool
	peerClient       CvrdtNetPeer

	factory PeerAppNodeModuleFactory

	PeerAppNodeModule
}

func (cc *ClusterClient) Init(ctx context.Context) {
	go cc.waitForMessages(ctx)
	cc.InitModule(ctx)
}

func (cc *ClusterClient) ShutDown() {

	if cc.clientActiveFlag.Load() {
		err := cc.peerClient.Shutdown()
		cc.Log("CvrdtClient %s was shut down with error: %v", cc.peerClient.Id.Format(), err)
	}

	err := cc.ShutDownModule()

	cc.Log("Connection with Cluster server %s closed with error: %v", cc.clusterInfo.Format(), err)

}

func (cc *ClusterClient) ConnectToCluster(ctx context.Context, newPeerAddress string) error {
	if cc.clientActiveFlag.Load() {
		return fmt.Errorf("Connection to cluster %s is already done", cc.clusterInfo.Format())
	}

	err := cc.Add(ctx, cc.clusterInfo)

	if err != nil {
		return err
	}

	responsePeers, err := cc.pingClusterServer(ctx, PING_JOIN, cc.Id, newPeerAddress)

	if err != nil {
		return err
	}

	if len(responsePeers) == 0 {
		return fmt.Errorf("Cluster server %s should return at least an assigned peer id", cc.clusterInfo.Format())
	}

	assignedInfo := responsePeers[0]
	otherPeersInfo := responsePeers[1:]

	domain := cc.clusterInfo.Name

	cc.peerClient = *CreateNewTCPWootBasedClient(assignedInfo, domain, cc.factory)

	go func() {
		cc.peerClient.Init(ctx)
	}()

	cc.clientActiveFlag.Store(true)

	cc.Log("CvrdtPeer %s created successfully", cc.peerClient.Id.Format())

	err = cc.peerClient.alignWithPeersData(ctx, otherPeersInfo)

	if err != nil {
		return err
	}

	return nil
}

func (cc *ClusterClient) DiffuseData(ctx context.Context) error {
	if !cc.clientActiveFlag.Load() {
		return fmt.Errorf("Connection to cluster %s is not executed yet", cc.clusterInfo.Format())
	}

	//ping cluster server to retrieve current cluster membership peers
	responsePeers, err := cc.pingClusterServer(ctx, PING_RETRIEVE, cc.Id, nil)

	if err != nil {
		return fmt.Errorf("error while trying to retrieve cluster server %s data: %v", cc.clusterInfo.Format(), err)
	}

	cc.Log("Cluster server retrieve returned %d peers", len(responsePeers))

	//align current cvrdtPeer knowledge of cluster with the new info provided info from the server
	err = cc.peerClient.alignWithPeersData(ctx, responsePeers)

	if err != nil {
		return err
	}

	//after the alignment data can be braodcasted
	return cc.peerClient.BroadcastRegistryOverTransport(ctx)
}

// ping cluster server with sender current hostinfo, support two topic ping: PING_RETRIEVE to retrieve peers and PING_JOIN to retrieve peers and join (with join also the peer address should be provided)
func (cc *ClusterClient) pingClusterServer(ctx context.Context, topic p2p.TopicName, senderHostInfo p2p.PeerHostInfo, payload any) ([]p2p.PeerHostInfo, error) {
	metadata := p2p.PeerMetadata{
		PeerHostInfo: senderHostInfo,
	}

	env, err := cc.Enveloper(metadata, payload)

	if err != nil {
		return []p2p.PeerHostInfo{}, err
	}

	session := cc.CreateNewSession(topic)
	defer session.Close()

	err = session.Broadcast(ctx, env) //Note: here the transport is related to cluster server, even using simple Send will work

	if err != nil {
		return []p2p.PeerHostInfo{}, err
	}

	ctxt, cancel := context.WithTimeout(ctx, CLUSTER_SERVER_RESPONSE_TIME)
	defer cancel()
	response, err := session.WaitOn(ctxt)

	if err != nil {
		return []p2p.PeerHostInfo{}, fmt.Errorf("Server did not respond: %v", err)
	}

	if response.Key.IsError != "" {
		return []p2p.PeerHostInfo{}, fmt.Errorf(response.Key.IsError)
	}

	var currentPeers []p2p.PeerHostInfo
	err = response.Data.Decode(&currentPeers)

	if err != nil {
		return []p2p.PeerHostInfo{}, err
	}

	return currentPeers, nil
}

func (cc *ClusterClient) CreateNewLocalOperation(filepath string, opStr string, content string, pos int) (bool, error) {
	if !cc.clientActiveFlag.Load() {
		return false, CVRDT_PEER_NOT_INITIALIZED
	}

	return cc.peerClient.MergeCvrdtOp(filepath, opStr, content, pos)

}

func (cc *ClusterClient) DisplayData(mode history.OpVisualizationMode, filepath string, maxDepth int) (string, error) {
	if !cc.clientActiveFlag.Load() {
		return "", CVRDT_PEER_NOT_INITIALIZED
	}

	if mode == history.ContentVisual {
		return cc.peerClient.VisualizeContent(filepath)
	}

	if mode == history.LinearVisual {
		return cc.peerClient.VisualizeHistory(filepath, false, 0)
	} else if mode == history.DagVisual {

		if maxDepth <= 0 {
			return "", fmt.Errorf("maxdepth should be greater than 0 to visualize at least a concurrency layer")
		}

		return cc.peerClient.VisualizeHistory(filepath, true, maxDepth)
	}

	return "", fmt.Errorf("mode argument should be an history.OpVisualizationMode")

}

func (cc *ClusterClient) waitForMessages(ctx context.Context) {

	cc.RegisterHandler(PEER_EXIT, func(ctx context.Context, session PeerSession, msg p2p.PeerMessage) {
		cc.Log("Connection with cluster server %s severed, Start shutting down client...", cc.clusterInfo.Format())
		cc.ShutDown()
	}).
		RegisterHandler(PEER_CONNECTION, func(ctx context.Context, session PeerSession, msg p2p.PeerMessage) {
			cc.Log("Connection with cluster server %s successfull!", cc.clusterInfo.Format())
		}).
		StartWaiting(ctx)
}

func CreateNewClusterClient(clusterInfo p2p.PeerHostInfo, id p2p.PeerHostInfo, factory PeerAppNodeModuleFactory) *ClusterClient {

	appNodeModule := factory.Create(id)

	return &ClusterClient{
		clusterInfo:       clusterInfo,
		PeerAppNodeModule: appNodeModule,
		factory:           factory,
	}
}
