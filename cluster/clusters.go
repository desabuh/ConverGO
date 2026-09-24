package cluster

import (
	"context"
	"errors"
	"fmt"

	"github.com/desabuh/convergo/p2p"
)

// var logFactory log.GlobalLoggerFactory = log.NewMutexLoggerFactory(config.LoggerDefaultConfigExtractor{})
// var networkModuleFactory p2p.NetworkModuleFactory[p2p.PeerMessage, p2p.PeerHostInfo, p2p.PeerMetadata] = p2p.TCPJsonPeerNetworkModuleFactory{ConfigExtractor: config.NeworkDefaultConfigExtractor{}}

// var appNodeModuleFactory cluster.AppNodeModuleFactory[p2p.PeerMessage, p2p.PeerHostInfo, p2p.PeerMetadata] = cluster.AppNodeModuleFactory[p2p.PeerMessage, p2p.PeerHostInfo, p2p.PeerMetadata]{NetworkModuleFactory: networkModuleFactory, LoggerFactory: logFactory}
type Node interface {
	GetId() p2p.PeerHostInfo
	Init(ctx context.Context)
	ShutDown()
}

type NodeKind string

const (
	NodeKindClient NodeKind = "client"
	NodeKindServer NodeKind = "server"
)

type ClusterNodeStore struct {
	nodes    map[p2p.PeerHostInfo]Node
	defaults map[NodeKind]p2p.PeerHostInfo
	factory  PeerAppNodeModuleFactory
}

func NewClusterNodeStore(factory PeerAppNodeModuleFactory) *ClusterNodeStore {
	return &ClusterNodeStore{
		nodes:    make(map[p2p.PeerHostInfo]Node),
		defaults: make(map[NodeKind]p2p.PeerHostInfo),
		factory:  factory,
	}
}

func GetNodeAs[T Node](id p2p.PeerHostInfo, node Node) (T, error) {
	nodeCasted, ok := node.(T)
	if !ok {
		return *new(T), fmt.Errorf("id %s is associated with the wrong type of node (client/server)", id.Format())
	}
	return nodeCasted, nil
}

func (s *ClusterNodeStore) init(ctx context.Context, node Node, kind NodeKind) error {
	id := node.GetId()

	_, ok := s.nodes[id]

	if ok {
		return fmt.Errorf("A node with id %s already exists", id.Format())
	}

	_, ok = s.defaults[kind]

	if ok {
		return fmt.Errorf("A node with kind %s is already defined in the store", kind)
	}

	s.nodes[id] = node
	s.defaults[kind] = node.GetId()

	go node.Init(ctx)

	return nil
}

func (s *ClusterNodeStore) getNode(id p2p.PeerHostInfo) (Node, error) {

	node, ok := s.nodes[id]

	if !ok {
		return nil, fmt.Errorf("no id is found in the node store")
	}

	return node, nil
}

func (s *ClusterNodeStore) getKindNode(kind NodeKind) (Node, error) {
	nodeInfo, ok := s.defaults[kind]

	if !ok {
		return nil, fmt.Errorf("no id is found in the defaults node store")
	}

	return s.getNode(nodeInfo)
}

func (s *ClusterNodeStore) CreateNewClient(ctx context.Context, clusterInfo p2p.PeerHostInfo, id p2p.PeerHostInfo) error {
	client := CreateNewClusterClient(clusterInfo, id, s.factory)
	return s.init(ctx, client, NodeKindClient)
}

func (s *ClusterNodeStore) CreateNewServer(ctx context.Context, id p2p.PeerHostInfo) error {
	server := CreateNewClusterServer(id, s.factory)
	return s.init(ctx, server, NodeKindServer)
}

func (s *ClusterNodeStore) GetClient() (*ClusterClient, error) {
	node, err := s.getKindNode(NodeKindClient)

	if err != nil {
		return nil, err
	}

	return GetNodeAs[*ClusterClient](node.GetId(), node)
}

func (s *ClusterNodeStore) GetServer() (*ClusterServer, error) {
	node, err := s.getKindNode(NodeKindServer)

	if err != nil {
		return nil, err
	}

	return GetNodeAs[*ClusterServer](node.GetId(), node)
}

func (s *ClusterNodeStore) Remove(id p2p.PeerHostInfo) error {
	node, err := s.getNode(id)
	if err != nil {
		return err
	}

	delete(s.nodes, id)
	for kind, defID := range s.defaults {
		if defID == id {
			delete(s.defaults, kind)
		}
	}

	//no error is returned by ShutDown (intended behavior, the node will log the error themselves)
	node.ShutDown()
	return nil
}

func (s *ClusterNodeStore) Close() error {

	ids := make([]p2p.PeerHostInfo, 0, len(s.nodes))
	for id := range s.nodes {
		ids = append(ids, id)
	}

	var err error
	for _, id := range ids {
		err = errors.Join(err, s.Remove(id))
	}
	return err
}
