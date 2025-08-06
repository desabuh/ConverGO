package p2p

type Message struct {
	PayLoad   string   `json:"Payload"`
	Protocoll string   `json:"Prot"`
	Sender    UserData `json:"UserData"`
}

type UserData struct {
	Id       string `json:"Id"`
	Username string `json:"Name"`
}
