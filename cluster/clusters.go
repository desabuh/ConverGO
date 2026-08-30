package cluster

import (
	"context"
	"errors"
	"fmt"
	"sync"

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
	mu       sync.Mutex
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

func (s *ClusterNodeStore) CreateNewClient(ctx context.Context, clusterInfo p2p.PeerHostInfo, id p2p.PeerHostInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	client := CreateNewClusterClient(clusterInfo, id, s.factory)
	return s.init(ctx, client)
}

func (s *ClusterNodeStore) CreateNewServer(ctx context.Context, id p2p.PeerHostInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	server := CreateNewClusterServer(id, s.factory)
	return s.init(ctx, server)
}

func (s *ClusterNodeStore) exists(node Node) bool {
	_, ok := s.nodes[node.GetId()]
	return ok
}

func (s *ClusterNodeStore) init(ctx context.Context, node Node) error {
	id := node.GetId()

	//s.mu.Lock()

	_, ok := s.nodes[id]

	if ok {
		return fmt.Errorf("A node with id %s already exists", id)
	}

	s.nodes[id] = node
	//s.mu.Unlock()

	go func() {
		defer s.Remove(id)
		node.Init(ctx)
	}()

	return nil
}

func (s *ClusterNodeStore) Remove(id p2p.PeerHostInfo) error {
	s.mu.Lock()
	node, ok := s.nodes[id]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("no id is found in the node store")
	}

	delete(s.nodes, id)
	for kind, defID := range s.defaults {
		if defID == id {
			delete(s.defaults, kind)
		}
	}
	s.mu.Unlock()

	node.ShutDown()
	return nil
}

func (s *ClusterNodeStore) GetDefaultClient() (*ClusterClient, error) {
	return s.getDefaultClient()
}

func (s *ClusterNodeStore) GetDefaultServer() (*ClusterServer, error) {
	return s.getDefaultServer()
}

func (s *ClusterNodeStore) SetDefaultClient(id p2p.PeerHostInfo) error {
	return s.setDefault(NodeKindClient, id)
}

func (s *ClusterNodeStore) SetDefaultServer(id p2p.PeerHostInfo) error {
	return s.setDefault(NodeKindServer, id)
}

func (s *ClusterNodeStore) setDefault(kind NodeKind, id p2p.PeerHostInfo) error {
	s.mu.Lock()
	node, ok := s.nodes[id]
	s.mu.Unlock()

	if !ok {
		return fmt.Errorf("cannot set default %s: no id associated with %s in store", kind, id.Format())
	}

	switch kind {
	case NodeKindClient:
		if _, ok := node.(*ClusterClient); !ok {
			return fmt.Errorf("cannot set default client: id %s is not a client", id.Format())
		}
	case NodeKindServer:
		if _, ok := node.(*ClusterServer); !ok {
			return fmt.Errorf("cannot set default server: id %s is not a server", id.Format())
		}
	default:
		return fmt.Errorf("unknown node kind %q", kind)
	}

	s.mu.Lock()
	s.defaults[kind] = id
	s.mu.Unlock()
	return nil
}

func (s *ClusterNodeStore) getDefaultID(kind NodeKind) (p2p.PeerHostInfo, error) {
	s.mu.Lock()
	id, ok := s.defaults[kind]
	s.mu.Unlock()

	if !ok {
		return p2p.PeerHostInfo{}, fmt.Errorf("no default %s was currently set", kind)
	}

	return id, nil
}

func (s *ClusterNodeStore) getNode(id p2p.PeerHostInfo) (Node, error) {
	s.mu.Lock()
	node, ok := s.nodes[id]
	s.mu.Unlock()

	if !ok {
		return nil, fmt.Errorf("no id is found in the node store")
	}

	return node, nil
}

func (s *ClusterNodeStore) GetClient(id p2p.PeerHostInfo) (*ClusterClient, error) {
	node, err := s.getNode(id)
	if err != nil {
		return nil, err
	}
	return GetNodeAs[*ClusterClient](id, node)
}

func (s *ClusterNodeStore) GetServer(id p2p.PeerHostInfo) (*ClusterServer, error) {
	node, err := s.getNode(id)
	if err != nil {
		return nil, err
	}
	return GetNodeAs[*ClusterServer](id, node)
}

func (s *ClusterNodeStore) getDefaultClient() (*ClusterClient, error) {
	id, err := s.getDefaultID(NodeKindClient)
	if err != nil {
		return nil, err
	}
	return s.GetClient(id)
}

func (s *ClusterNodeStore) getDefaultServer() (*ClusterServer, error) {
	id, err := s.getDefaultID(NodeKindServer)
	if err != nil {
		return nil, err
	}
	return s.GetServer(id)
}

func (s *ClusterNodeStore) Close() error {
	s.mu.Lock()
	ids := make([]p2p.PeerHostInfo, 0, len(s.nodes))
	for id := range s.nodes {
		ids = append(ids, id)
	}
	s.mu.Unlock()

	var err error
	for _, id := range ids {
		err = errors.Join(err, s.Remove(id))
	}
	return err
}

func GetNodeAs[T Node](id p2p.PeerHostInfo, node Node) (T, error) {
	nodeCasted, ok := node.(T)
	if !ok {
		return *new(T), fmt.Errorf("id %s is associated with the wrong type of node (client/server)", id.Format())
	}
	return nodeCasted, nil
}
