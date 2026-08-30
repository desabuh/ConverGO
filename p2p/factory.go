package p2p

import (
	"github.com/desabuh/convergo/config"
)

// a generic factory to return a new NetworkModule[M, T, R], provide two methods:
// - Create accept the module identifier T
// - CreateFromArgs provide other that the identifier also additional arguments to override the factory default behavior
// Should also implement a ConfigExtractor interface to extract configuration from the hashmap
type NetworkModuleFactory[M comparable, T comparable, R comparable] interface {
	Create(id T) NetworkModule[M, T, R]
	CreateFromArgs(id T, args map[string]any) NetworkModule[M, T, R]
	config.ConfigExtractor[config.NetworkModuleConfig]
}

type NetworkModule[M comparable, T comparable, R comparable] struct {
	Transport[M, T, R]
	MessageBroker[M, T]
	Enveloper[R]
}

// a factory that return a new NetworkModule based on TCP transport, json based encoding and PeerMessage based communication
type TCPJsonPeerNetworkModuleFactory struct {
	config.ConfigExtractor[config.NetworkModuleConfig]
}

func (nm TCPJsonPeerNetworkModuleFactory) Create(id PeerHostInfo) NetworkModule[PeerMessage, PeerHostInfo, PeerMetadata] {

	networkConfig := nm.ExtractFrom(map[string]any{})

	return nm.initModule(id, networkConfig)

}

func (nm TCPJsonPeerNetworkModuleFactory) CreateFromArgs(id PeerHostInfo, args map[string]any) NetworkModule[PeerMessage, PeerHostInfo, PeerMetadata] {

	networkConfig := nm.ExtractFrom(args)

	return nm.initModule(id, networkConfig)

}

func (nm TCPJsonPeerNetworkModuleFactory) initModule(id PeerHostInfo, networkConfig config.NetworkModuleConfig) NetworkModule[PeerMessage, PeerHostInfo, PeerMetadata] {

	jsonTransportCodec := NewCodec(JsonEnvelopeEncoder[PeerMetadata]{}, JsonEnvelopeDecoder[PeerMetadata]{})

	var clusterTransport = NewTCPTransportWithHostExhange(id, jsonTransportCodec)
	clusterTransport.SetCodec(jsonTransportCodec)

	var jsonEnveloper Enveloper[PeerMetadata] = func(meta PeerMetadata, payload any) (Envelope[PeerMetadata], error) {
		if payload == nil {
			return GetEmptyEnvelope(meta), nil
		} else {
			return GetJsonBasedEnvelope(meta, payload)
		}
	}

	clusterTransport.SetLocalMsgFactory(jsonEnveloper)

	broker := NewPeerMessageBroker(clusterTransport)

	config.WithConfig(clusterTransport, networkConfig.TransportConfig)
	config.WithConfig(broker, networkConfig.BrokerConfig)

	return NetworkModule[Envelope[PeerMetadata], PeerHostInfo, PeerMetadata]{
		Transport:     clusterTransport,
		MessageBroker: broker,
		Enveloper:     jsonEnveloper,
	}

}
