package cluster

import "github.com/desabuh/convergo/p2p"

//simpler generic type alias imports to instance AppNodeModuleFactory[p2p.PeerMessage, p2p.PeerHostInfo, p2p.PeerMetadata]
type PeerAppNodeModuleFactory = AppNodeModuleFactory[p2p.PeerMessage, p2p.PeerHostInfo, p2p.PeerMetadata]

//simpler generic type alias imports to instance AppNodeModule[p2p.PeerMessage, p2p.PeerHostInfo, p2p.PeerMetadata]
type PeerAppNodeModule = AppNodeModule[p2p.PeerMessage, p2p.PeerHostInfo, p2p.PeerMetadata]

//simpler generic type alias imports to instance p2p.Session[p2p.PeerMessage, p2p.PeerHostInfo]
type PeerSession = p2p.Session[p2p.PeerMessage, p2p.PeerHostInfo]
