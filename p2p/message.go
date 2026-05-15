package p2p

import (
	"fmt"
	"net"
)

type PeerMessage = Envelope[PeerMetadata]

type PeerMetadata struct {
	MessageName string
	isError     string
	PeerHostInfo
}

func GetPeerMetadata(name string, isError string, info PeerHostInfo) PeerMetadata {
	return PeerMetadata{
		MessageName:  name,
		isError:      isError,
		PeerHostInfo: info,
	}
}

type PeerHostInfo struct {
	Id string
	UserData
}

func (info PeerHostInfo) isPeerClosed() bool {
	return info.Address == nil
}

func (info PeerHostInfo) Format() string {

	var addr = info.Address.String()

	if info.isPeerClosed() {
		addr = "Vacant"
	}

	return fmt.Sprintf("%s_%s_%s", info.Id, info.Name, addr)
}

func CreateNewHostInfo(id string, name string, address string) (PeerHostInfo, error) {

	netAddr, err := net.ResolveTCPAddr("tcp", address)

	if err != nil {
		return PeerHostInfo{}, fmt.Errorf("Cannot create a new host info: %v", err)
	}

	return PeerHostInfo{
		Id: id,
		UserData: UserData{
			Name:    name,
			Address: netAddr,
		},
	}, nil

}

type UserData struct {
	Name    string
	Address *net.TCPAddr
}
