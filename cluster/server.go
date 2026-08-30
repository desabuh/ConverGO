package cluster

import (
	"context"
	"errors"
	"fmt"

	"github.com/desabuh/convergo/p2p"
)

var (
	ErrUserAlreadyExists = errors.New("User already exists")
	ErrUserPayload       = errors.New("User payload was provided in the wrong format")
)

type ClusterServer struct {
	PeerAppNodeModule

	ClusterStore
}

func (cs *ClusterServer) Init(ctx context.Context) {
	go cs.waitForMessages(ctx)
	cs.InitModule(ctx)

}

func (cs *ClusterServer) ShutDown() {

	err := cs.ShutDownModule()

	cs.Log("Cluster server shut down with error: %v", err)

}

func (cs *ClusterServer) waitForMessages(ctx context.Context) {

	cs.RegisterHandler(PING_RETRIEVE, func(ctx context.Context, session PeerSession, msg p2p.PeerMessage) {

		cs.Log("Received retrieve request from %s", msg.Key.PeerHostInfo.Format())

		metadata := p2p.PeerMetadata{
			PeerHostInfo: cs.Id,
		}

		data := cs.RetrieveCurrentPeers()

		env, err := cs.Enveloper(metadata, data)

		if err != nil {
			cs.Log("Cannot serialize data for %s: %v", session.GetTopic(), err)
			return
		}

		err = session.Send(ctx, env, msg.Key.PeerHostInfo)

		if err != nil {
			cs.Log("Failure to respond to %s: %v", msg.Key.PeerHostInfo.Format(), err)
			return
		}

		cs.Log("Respond to %s with current peers (num. %d)", msg.Key.PeerHostInfo.Format(), len(data))

	}).
		RegisterHandler(PING_JOIN, func(ctx context.Context, session PeerSession, msg p2p.PeerMessage) {

			sendError := func(err error) {
				metadata := p2p.PeerMetadata{
					IsError:      err.Error(),
					PeerHostInfo: cs.Id,
				}

				res, err := cs.Enveloper(metadata, nil)

				if err != nil {
					session.Send(ctx, res, msg.Key.PeerHostInfo)
				}
			}

			var info p2p.PeerHostInfo = msg.Key.PeerHostInfo

			var peerAddress string
			if err := msg.Data.Decode(&peerAddress); err != nil {
				sendError(fmt.Errorf("%w: %v", ErrUserPayload, err))
				return
			}

			resultPeers, err := cs.AddNewPeer(info, peerAddress)
			if err != nil {
				sendError(ErrUserAlreadyExists)
				return
			}

			cs.Log("A new peer %s was registered", resultPeers[0].Format())

			metadata := p2p.PeerMetadata{
				PeerHostInfo: cs.Id,
			}

			res, err := cs.Enveloper(metadata, resultPeers)

			if err != nil {
				cs.Log("Cannot serialize data for %s: %v", session.GetTopic(), err)
				return
			}

			session.Send(ctx, res, info)

		}).
		RegisterHandler(PEER_EXIT, func(ctx context.Context, session PeerSession, msg p2p.PeerMessage) {

			info := msg.Key.PeerHostInfo

			peerInfo, removed := cs.RemovePeer(info)

			if removed {
				cs.Log("%s peer left the cluster", peerInfo.Format())
			}

		}).
		StartWaiting(ctx)

}

func CreateNewClusterServer(info p2p.PeerHostInfo, factory PeerAppNodeModuleFactory) *ClusterServer {

	nodeModule := factory.Create(info)

	return &ClusterServer{
		PeerAppNodeModule: nodeModule,
		ClusterStore:      ClusterStore{clientCreatedPeers: make(map[p2p.UserData]p2p.PeerHostInfo)},
	}

}
