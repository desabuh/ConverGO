package cluster

import (
	"context"
	"errors"
	"fmt"

	"github.com/desabuh/convergo/cvrdt"
	"github.com/desabuh/convergo/file/repository"
	"github.com/desabuh/convergo/p2p"
	"github.com/desabuh/convergo/utils"
)

const (
	PEER_SESSION_MESSAGE_QUEUE_MAX_SIZE = 20
)

type CvrdtNetPeer struct {
	PeerAppNodeModule

	repo repository.CvrdtRepository

	crdtAdapter repository.CvrdtToRepoFilesAdapter[p2p.Payload, cvrdt.WootOperation]
}

func (c *CvrdtNetPeer) Shutdown() error {
	var errs []error

	if err := c.ShutDownModule(); err != nil {
		errs = append(errs, err)
	}

	if err := c.repo.ShutDownRepo(); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func (c *CvrdtNetPeer) Pair(ctx context.Context, target p2p.PeerHostInfo) error {
	err := c.Add(ctx, target)

	if err != nil && !errors.Is(err, p2p.ErrPeerAlreadyExists) {
		return err
	}

	return nil
}

func (c *CvrdtNetPeer) DisconnectFrom(target p2p.PeerHostInfo) error {
	err := c.Remove(target)

	if err != nil && !errors.Is(err, p2p.ErrPeerDoesNotExists) {
		return err
	}

	return nil
}

func (c *CvrdtNetPeer) BroadcastRegistryOverTransport(ctx context.Context) error {

	metadata := p2p.PeerMetadata{
		PeerHostInfo: c.Id,
	}

	env, err := c.Enveloper(metadata, c.repo.GetRepoFiles())

	if err != nil {
		return err
	}

	session := c.CreateNewSession(PUSH)
	defer session.Close()

	return session.Broadcast(ctx, env)

}

func (c *CvrdtNetPeer) alignWithPeersData(ctx context.Context, newPeerList []p2p.PeerHostInfo) error {

	newPeersSet := make(map[p2p.PeerHostInfo]struct{}, len(newPeerList))
	currentPeersSet := c.GetTargets()

	for _, v := range newPeerList { //for each peer not present in the current list, pair them
		if c.Id == v { //ignore local host info
			continue
		}

		newPeersSet[v] = struct{}{}

		if _, ok := currentPeersSet[v]; !ok {
			err := c.Pair(ctx, v)

			if err != nil {
				return err
			}
		}
	}

	for v := range currentPeersSet { // for each peer not present in the new list, remove them
		if _, ok := newPeersSet[v]; !ok {

			err := c.DisconnectFrom(v)

			if err != nil {
				return err
			}
		}
	}

	return nil

}

func (c *CvrdtNetPeer) MergeCvrdtOp(filepath string, opTypeStr string, content string, pos int) (bool, error) {

	var opType cvrdt.OpType
	if opTypeStr == "insert" {
		opType = cvrdt.Insertion
	} else if opTypeStr == "delete" {
		opType = cvrdt.Deletion
	} else {
		return false, fmt.Errorf("Supported operation modes are 'insert' or 'delete' not %s", opTypeStr)
	}

	newState := cvrdt.GetNewStateFromOp(
		cvrdt.CreateNewLocalOp(opType, pos, content, c.Id.Id),
	)

	newInfo := repository.CvrdtFileInfo{
		LocalPath:     filepath,
		ReadOnlyState: newState,
	}

	updatedInfo, err := c.repo.UpdateFileCtx(newInfo)

	if err != nil {
		return false, fmt.Errorf("error while trying to merge new cvrdt operation on %s context: %v", filepath, err)
	}

	if updatedInfo.IsNewlyCreated {
		c.Log("File context %s was created", filepath)
	}

	if opType == cvrdt.Deletion {
		content = updatedInfo.ReadOnlyState[len(updatedInfo.ReadOnlyState)-1].Char()
	}

	c.Log("Merged local %s operation of %s in position %d for file context %s", opType, content, pos, filepath)

	return updatedInfo.IsNewlyCreated, err
}

func (c *CvrdtNetPeer) VisualizeHistory(filepath string, asDag bool, maxDepth int) (string, error) {

	args := map[string]any{}

	if asDag {
		args["depth"] = maxDepth
		return c.repo.DisplayFileHistory(filepath, "dag", args)
	}

	return c.repo.DisplayFileHistory(filepath, "linear", args)

}

func (c *CvrdtNetPeer) VisualizeContent(filepath string) (string, error) {
	content, err := c.repo.GetFileContent(filepath)

	if err != nil {
		return "", err
	}

	return content, nil
}

func (c *CvrdtNetPeer) waitForMessages(ctx context.Context) {

	c.RegisterHandler(PUSH,
		func(ctx context.Context, session PeerSession, msg p2p.PeerMessage) {
			c.Log("PUSH request received from %s", msg.Key.PeerHostInfo.Format())

			//contexts, err := c.DecodeFileContexts(msg.Data)
			contexts, err := c.crdtAdapter.DecodeToRepoFiles(msg.Data)

			if err != nil {
				c.Log("PUSH request payload from %s cannot be unwrapped %v, message discarded", msg.Key.PeerHostInfo.Format(), err)
				return
			}

			c.Log("PUSH request received from %s for %d file contexts, start merging:", msg.Key.PeerHostInfo.Format(), len(contexts))

			for _, newCtx := range contexts {

				_, err := c.repo.UpdateFileCtx(newCtx)

				if err != nil {
					c.Log("File context %s was not merged correctly, discard remaining ctxs and stopping PUSH operation: %v", newCtx.LocalPath, err)
					return
				}

				c.Log("File context %s with %d operations successfully merged!", newCtx.LocalPath, len(newCtx.ReadOnlyState))

			}

			c.Log("PUSH request successfull!")

		}).
		RegisterHandler(PEER_CONNECTION,
			func(ctx context.Context, session PeerSession, msg p2p.PeerMessage) {

				c.Log("Pairing request from CvrdtClient %s was successfull!, broadcast current state to everyone:", msg.Key.PeerHostInfo.Format())

				err := c.BroadcastRegistryOverTransport(ctx)

				if err != nil {
					c.Log("Push broadcasting failure: %v", err)
				} else {
					c.Log("Push broadcasting was a success!")
				}

			}).
		RegisterHandler(PEER_EXIT,
			func(ctx context.Context, session PeerSession, msg p2p.PeerMessage) {
				c.Log("Peer %s disconnected, reason: %s", msg.Key.PeerHostInfo.Format(), msg.Key.IsError)
			}).
		StartWaiting(ctx)
}

func (c *CvrdtNetPeer) Init(ctx context.Context) {
	go c.waitForMessages(ctx)
	c.InitModule(ctx)
}

func CreateNewTCPWootBasedClient(peerInfo p2p.PeerHostInfo, domainPath string, factory PeerAppNodeModuleFactory) *CvrdtNetPeer {

	//note: cvrdtclient could need a slightly larger session queue cause of the client have to manage a larger number of message received
	appNodeModule := factory.CreateFromArgs(peerInfo, map[string]any{
		"sessionQueueSize": PEER_SESSION_MESSAGE_QUEUE_MAX_SIZE,
	})

	CRDTfactory := func() utils.ObservableState[cvrdt.CvRDTState, string] {
		return cvrdt.NewWootCvrdtWithView(peerInfo.Id)
	}

	return &CvrdtNetPeer{
		PeerAppNodeModule: appNodeModule,
		repo:              *repository.GetNewCvrdtRepository(peerInfo.Id, domainPath, CRDTfactory),
	}

	// return &CvrdtNetPeer{
	// 	PeerAppNodeModule: appNodeModule,
	// 	registry:          *file.CreateNewWootFileRegistry(peerInfo.Id, domainPath),
	// }
}
