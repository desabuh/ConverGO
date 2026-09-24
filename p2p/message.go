package p2p

import (
	"fmt"
	"strings"
)

const (
	CLUSTER_CLIENT_ID = "Client"
	CLUSTER_SERVER_ID = "Server"
)

type PeerMessage = Envelope[PeerMetadata]

type PeerMetadata struct {
	IsError string
	PeerHostInfo
	SessionFields
}

func GetPeerMetadata(topic TopicName, isError string, info PeerHostInfo) PeerMetadata {
	return PeerMetadata{
		SessionFields: SessionFields{
			TopicName: topic,
		},
		IsError:      isError,
		PeerHostInfo: info,
	}
}

type PeerHostInfo struct {
	Id string
	UserData
}

func (info PeerHostInfo) Format() string {

	return fmt.Sprintf("%s_%s_%s", info.Id, info.Name, info.Address)
}

func CreateNewHostInfo(id string, name string, address string) PeerHostInfo {

	return PeerHostInfo{
		Id: id,
		UserData: UserData{
			Name:    name,
			Address: address,
		},
	}

}

func ParseHostInfoFromStr(infoStr string, specialId string) (PeerHostInfo, error) {
	parts := strings.Split(infoStr, "_")

	if len(parts) == 3 {
		return CreateNewHostInfo(parts[0], parts[1], parts[2]), nil
	}

	if len(parts) == 2 && (specialId == CLUSTER_CLIENT_ID || specialId == CLUSTER_SERVER_ID) {

		userData := UserData{Name: parts[0], Address: parts[1]}

		switch specialId {
		case CLUSTER_CLIENT_ID:
			return CreateClientInfo(userData), nil
		case CLUSTER_SERVER_ID:
			return CreateServerInfo(userData), nil
		}

	}

	return PeerHostInfo{}, fmt.Errorf("peerhostinfo should be in form <id>_<name>_<address>")
}

func GenerateInfoFromUserData(id string, info PeerHostInfo) PeerHostInfo {
	return PeerHostInfo{
		Id:       id,
		UserData: info.UserData,
	}
}

func CreateClientInfo(user UserData) PeerHostInfo {
	return GetInfoFromUserData(CLUSTER_CLIENT_ID, user)
}

func CreateServerInfo(user UserData) PeerHostInfo {
	return GetInfoFromUserData(CLUSTER_SERVER_ID, user)
}

func GetInfoFromUserData(id string, user UserData) PeerHostInfo {
	return CreateNewHostInfo(id, user.Name, user.Address)
}

type UserData struct {
	Name    string
	Address string
}
