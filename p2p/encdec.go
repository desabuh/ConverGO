package p2p

import (
	"encoding/json"
	"fmt"
)

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

////////////////////////////////////

func GetJsonBasedEnvelope[K comparable](key K, data any) (Envelope[K], error) {

	payload, err := NewJsonPayloadEncode(data)

	if err != nil {
		return Envelope[K]{}, err
	}

	return Envelope[K]{
		Key:  key,
		Data: payload,
	}, nil
}

type JsonPayload struct {
	raw json.RawMessage
}

func NewJsonPayload(data []byte) *JsonPayload {
	return &JsonPayload{raw: data}
}

func NewJsonPayloadEncode(data any) (JsonPayload, error) {
	byte, err := json.Marshal(data)

	if err != nil {
		return JsonPayload{}, err
	}

	return JsonPayload{raw: byte}, err

}

func (p JsonPayload) Decode(v any) error {
	return json.Unmarshal(p.raw, v)
}

type jsonEnvelope[K any] struct {
	Key  K               `json:"key"`
	Data json.RawMessage `json:"data"`
}

type JsonEnvelopeDecoder[K comparable] struct{}

func (d JsonEnvelopeDecoder[K]) Decode(data []byte) (Envelope[K], error) {
	var tmp jsonEnvelope[K]

	if err := json.Unmarshal(data, &tmp); err != nil {
		var zero Envelope[K]
		return zero, err
	}

	return Envelope[K]{
		Key:  tmp.Key,
		Data: NewJsonPayload(tmp.Data),
	}, nil
}

type JsonEnvelopeEncoder[K comparable] struct{}

func (e JsonEnvelopeEncoder[K]) Encode(env Envelope[K]) ([]byte, error) {
	// We assume Data is already JSON-compatible
	// If needed, you can require a method like `Raw() []byte`

	var raw json.RawMessage

	switch p := env.Data.(type) {
	case JsonPayload:
		raw = p.raw
	default:
		return nil, fmt.Errorf("unsupported payload type")
	}

	return json.Marshal(struct {
		Key  K               `json:"key"`
		Data json.RawMessage `json:"data"`
	}{
		Key:  env.Key,
		Data: raw,
	})
}

func NewJsonEnvelopeCodec[K comparable]() Codec[Envelope[K]] {
	return NewCodec[Envelope[K]](
		JsonEnvelopeEncoder[K]{},
		JsonEnvelopeDecoder[K]{},
	)
}

// // a partial decoder that first partially decode an envelope to gain its identifier K and after that, use K to choose the O in which decode the byte payload
// type PartialCondDecoder[K comparable, O any] struct {
// 	envelopeDecoder Decoder[Envelope[K]]
// 	payloadDecoders map[K]Decoder[O]
// }

// func (d *PartialCondDecoder[K, O]) Decode(data []byte) (O, error) {
// 	var zero O

// 	env, err := d.envelopeDecoder.Decode(data)
// 	if err != nil {
// 		return zero, err
// 	}

// 	dec, ok := d.payloadDecoders[env.Key]
// 	if !ok {
// 		return zero, fmt.Errorf("unknown key: %v", env.Key)
// 	}

// 	return dec.Decode(env.Payload)
// }

// type PartialCondEncoder[K comparable, I any] struct {
// 	envelopeEncoder Encoder[Envelope[K]]
// 	payloadEncoders map[K]Encoder[I]
// 	keySelector     func(I) K
// }

// func (e *PartialCondEncoder[K, I]) Encode(msg I) ([]byte, error) {
// 	key := e.keySelector(msg)

// 	enc, ok := e.payloadEncoders[key]
// 	if !ok {
// 		return nil, fmt.Errorf("no encoder for key: %v", key)
// 	}

// 	payload, err := enc.Encode(msg)
// 	if err != nil {
// 		return nil, err
// 	}

// 	env := Envelope[K]{
// 		Key:     key,
// 		Payload: payload,
// 	}

// 	return e.envelopeEncoder.Encode(env)
// }

// type MessageType string

// const (
// 	MsgHello MessageType = "hello"
// 	MsgData  MessageType = "data"
// )

// type Hello struct {
// 	Name string
// }

// type Data struct {
// 	Value int
// }

// func CreateNewPartialJsonEncoder[K comparable, I any]() PartialCondDecoder[K, I] {
// 	return PartialCondDecoder[K, I]{
// 		envelopeDecoder: JsonDecoder[Envelope[K]]{},
// 	}
// }
