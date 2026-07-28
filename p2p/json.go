package p2p

import (
	"encoding/json"
	"fmt"
)

func NewJsonCodec[T any]() Codec[T] {
	return NewCodec(JsonEncoder[T]{}, JsonDecoder[T]{})
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
	case EmptyPayload: //if payload was specified as empty substitute an empty json payload
		raw = []byte("{}")
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
