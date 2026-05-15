package p2p

import "fmt"

// a payload it's the data content of a message, it provide a v argument to decode in place the payload content
// It should only be used by a server with knowledge about the resulting type of the payload
// No assumption is made about current payload state (e.g. byte or intermediate structure)
// passing a non matching type should throw an error
type Payload interface {
	Decode(v any) error
}

// an envelope is a partial decoded message with a K clear identifier data and a payload to decode
// Generally K should be used as context to understand Payload management
type Envelope[K comparable] struct {
	Key  K
	Data Payload
}

func GetEmptyEnvelope[K comparable](metadata K) Envelope[K] {
	return Envelope[K]{
		Key:  metadata,
		Data: EmptyPayload{},
	}
}

type EmptyPayload struct{}

func (p EmptyPayload) Decode(v any) error {
	return fmt.Errorf("decoded payload is empty")
}

type Encoder[I any] interface {
	Encode(msg I) ([]byte, error)
}

type Decoder[O any] interface {
	Decode(data []byte) (O, error)
}

type Codec[ED any] interface {
	Encoder[ED]
	Decoder[ED]
}

func NewCodec[T any](encoder Encoder[T], decoder Decoder[T]) Codec[T] {
	return struct {
		Encoder[T]
		Decoder[T]
	}{
		Encoder: encoder,
		Decoder: decoder,
	}
}
