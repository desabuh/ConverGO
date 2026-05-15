package p2p

import (
	"context"
	"net"
	"os"
	"time"

	"github.com/desabuh/convergo/cvrdt"
	"github.com/desabuh/convergo/file"
	"github.com/desabuh/convergo/utils"
)

// a peer client, that contains a transport layer for communication and a registry to mantain internal data
// - D: the data to send over the transport layer
// - T: the identification of entities that communicates over the transport
// - M: the state type managed into the registry
type Client[I comparable, D any, T comparable, M utils.Clonable[M], R any] struct {
	Id        I
	transport Transport[D, T, R]
	Registry  file.FileRegistry[M]
}

// a client peer with CvRDT capabilities
type CvrdtNetClient Client[PeerHostInfo, PeerMessage, net.Addr, cvrdt.CvRDTState, PeerMetadata] //to do, use PeerHostInfo instead of net.Addr

func (c *CvrdtNetClient) BroadcastRegistryOverTransport(ctx context.Context) error {

	metadata := PeerMetadata{
		MessageName:  "PUSH",
		PeerHostInfo: c.Id,
	}

	env, err := GetJsonBasedEnvelope(metadata, c.Registry.GetFileRegistryInfo())

	if err != nil {
		return err
	}

	return c.transport.Broadcast(ctx, env)
}

func (c *CvrdtNetClient) WaitForMessages(ctx context.Context) {

	recCh := c.transport.GetReceiveCh()

	for { // <- Continuously listen
		select {
		case env := <-recCh:
			c.parseRequests(env)
		case <-ctx.Done():
			return
		}
	}
}

func (c *CvrdtNetClient) parseRequests(message PeerMessage) {
	switch message.Key.MessageName {
	case "PING_HANDSHAKE_SUCCESS":
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		utils.Logger.NlLog(os.Stdout, "CvrdtClient %s was paired with success!, broadcast current state to everyone:", message.Key.PeerHostInfo.Format())

		err := c.BroadcastRegistryOverTransport(ctx)

		if err != nil {
			utils.Logger.NlLog(os.Stdout, "Push broadcasting failure: %v", err)
		} else {
			utils.Logger.NlLog(os.Stdout, "Push broadcasting was a success!")
		}

	case "PUSH":

		utils.Logger.NlLog(os.Stdout, "PUSH request received from %s", message.Key.PeerHostInfo.Format())

		var res []file.FileContextInfo[[]cvrdt.WootOperation]

		err := message.Data.Decode(&res)

		if err != nil {
			utils.Logger.NlLog(os.Stdout, "PUSH request payload from %s cannot be unwrapped %v, message discarded", message.Key.PeerHostInfo.Format(), err)
			return
		}

		var contexts = utils.Map(res, func(x file.FileContextInfo[[]cvrdt.WootOperation]) file.FileContextInfo[cvrdt.CvRDTState] {
			result := make(cvrdt.CvRDTState, len(x.ReadOnlyState))

			for i, op := range x.ReadOnlyState {
				result[i] = op
			}

			return file.FileContextInfo[cvrdt.CvRDTState]{
				LocalPath:     x.LocalPath,
				ReadOnlyState: result,
			}
		})

		utils.Logger.NlLog(os.Stdout, "PUSH request received from %s for %d file contexts, start merging:", message.Key.PeerHostInfo.Format(), len(contexts))

		for _, newCtx := range contexts {

			_, err := c.Registry.UpdateFileCtx(newCtx)

			if err != nil {
				utils.Logger.NlLog(os.Stdout, "File context %s was not merged correctly, stopping PUSH operation: %v", newCtx.LocalPath, err)
				return
			}

			utils.Logger.NlLog(os.Stdout, "File context %s with %d operations successfully merged!", newCtx.LocalPath, len(newCtx.ReadOnlyState))

		}

		utils.Logger.NlLog(os.Stdout, "PUSH request successfull!")
	}
}

func (c *CvrdtNetClient) Init() error {
	return c.transport.ListenFor(c.Id.Address)
}

func (c *CvrdtNetClient) Pair(ctx context.Context, strAddr string) error {

	targetAddr, err := net.ResolveTCPAddr("tcp", strAddr)

	if err != nil {
		return err
	}

	return c.transport.Add(ctx, targetAddr)
}

func CreateNewTCPWootBasedClient(peerInfo PeerHostInfo, domainPath string, handshakeCodec Codec[PeerHostInfo], codec Codec[PeerMessage]) *CvrdtNetClient {
	var registry *file.FileRegistry[cvrdt.CvRDTState] = file.CreateNewWootFileRegistry(peerInfo.Id, domainPath)

	var network *TCPLayer[PeerMessage] = NewTCPTransportWithHostExhange[PeerMessage](peerInfo, handshakeCodec)
	network.SetCodec(codec)

	var emptyEnvFactoryFunc = func(x PeerMetadata) Envelope[PeerMetadata] { return GetEmptyEnvelope[PeerMetadata](x) }

	network.SetLocalPingFactory(emptyEnvFactoryFunc)

	return &CvrdtNetClient{
		Id:        peerInfo,
		transport: network,
		Registry:  *registry,
	}
}
