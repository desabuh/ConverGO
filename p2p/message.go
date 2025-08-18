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

func NewMatchingEncDec[T any](encoder Encoder[T], decoder Decoder[T]) EncoderDecoder[T] {
	return struct {
		Encoder[T]
		Decoder[T]
	}{
		Encoder: encoder,
		Decoder: decoder,
	}
}

type JsonEncoder[T any] struct{}

func (e JsonEncoder[T]) Encode(data T) ([]byte, error) {
	return json.Marshal(data)
}

type JsonDecoder[T any] struct{}

func (d JsonDecoder[T]) Decode(data []byte) (T, error) {
	var msg T
	err := json.Unmarshal(data, &msg)
	return msg, err
}
