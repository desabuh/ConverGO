package p2p

import (
	"encoding/json"

	"github.com/google/uuid"
)

type Message struct {
	Name string         `json:"Name"`
	Args map[string]any `json:"Args"`
	Time int64          `json:"Time"`
}

type UserData struct {
	Id       uuid.UUID `json:"Id"`
	Username string    `json:"Name"`
}

type Encoder interface {
	Encode(msg Message) ([]byte, error)
}

type Decoder interface {
	Decode(data []byte) (Message, error)
}

type JsonEncoder struct{}

func (e JsonEncoder) Encode(msg Message) ([]byte, error) {
	return json.Marshal(msg)
}

type JsonDecoder struct{}

func (d JsonDecoder) Decode(data []byte) (Message, error) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	return msg, err
}
