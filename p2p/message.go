package p2p

import (
	"encoding/json"

	"github.com/google/uuid"
)

type Message struct {
	Name string
	Args map[string]any
	Time int64
}

type UserData struct {
	Id       uuid.UUID
	Username string
}

type Encoder[I any] interface {
	Encode(msg I) ([]byte, error)
}

type Decoder[O any] interface {
	Decode(data []byte) (O, error)
}

type EncoderDecoder[ED any] interface {
	Encoder[ED]
	Decoder[ED]
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
