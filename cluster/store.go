package cluster

import (
	"fmt"
	"slices"
	"sync"

	"github.com/desabuh/convergo/p2p"
)

type ClusterStore struct {
	mu                 sync.RWMutex
	autoIncId          uint64
	clientCreatedPeers map[p2p.UserData]p2p.PeerHostInfo
}

// create a new peerhostinfo and prepend it to the whole list of info before returning it, also provide the address where the new peer will be created
func (cs *ClusterStore) AddNewPeer(info p2p.PeerHostInfo, peerAddress string) ([]p2p.PeerHostInfo, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if cs.isUserExists(info.UserData) {
		return []p2p.PeerHostInfo{}, ErrUserAlreadyExists
	}

	cs.autoIncId++

	newPeerInfo := p2p.CreateNewHostInfo(fmt.Sprint(cs.autoIncId), info.Name, peerAddress)

	cs.clientCreatedPeers[info.UserData] = newPeerInfo

	return slices.Insert(cs.retrieveLockedCurrentPeers(), 0, newPeerInfo), nil

}

func (cs *ClusterStore) RetrieveCurrentPeers() []p2p.PeerHostInfo {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	return cs.retrieveLockedCurrentPeers()
}

func (cs *ClusterStore) RemovePeer(info p2p.PeerHostInfo) (p2p.PeerHostInfo, bool) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if peerInfo, ok := cs.clientCreatedPeers[info.UserData]; ok {
		delete(cs.clientCreatedPeers, info.UserData)

		return peerInfo, true
	}

	return p2p.PeerHostInfo{}, false

}

func (cs *ClusterStore) isUserExists(user p2p.UserData) bool {
	_, ok := cs.clientCreatedPeers[user]
	return ok

}

func (cs *ClusterStore) retrieveLockedCurrentPeers() []p2p.PeerHostInfo {

	peerSlice := make([]p2p.PeerHostInfo, 0, len(cs.clientCreatedPeers))
	for _, peerInfo := range cs.clientCreatedPeers {
		peerSlice = append(peerSlice, peerInfo)
	}

	return peerSlice
}
