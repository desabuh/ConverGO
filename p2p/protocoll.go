package p2p

type ProtocollHandler interface {
	InterpretPayload(msg Message) (any, error)
}

var protocolRegistry = make(map[string]ProtocollHandler)

func RegisterProtocol(name string, handler ProtocollHandler) {
	protocolRegistry[name] = handler
}

// type Encoder struct {
// 	writer io.Writer
// }

// func NewEncoder(w io.Writer) *Encoder {
// 	return &Encoder{writer: w}
// }

// func (e *Encoder) Encode(msg Message) error {
// 	data, err := json.Marshal(msg)
// 	if err != nil {
// 		return err
// 	}

// 	data = append(data, '\n')
// 	_, err = e.writer.Write(data)
// 	return err
// }

// type Decoder struct {
// 	reader *bufio.Reader
// }

// func NewDecoder(r io.Reader) *Decoder {
// 	return &Decoder{reader: bufio.NewReader(r)}
// }

// func (d *Decoder) Decode() (Message, any, error) {
// 	line, err := d.reader.ReadBytes('\n')
// 	if err != nil {
// 		return Message{}, nil, err
// 	}

// 	var msg Message
// 	if err := json.Unmarshal(line, &msg); err != nil {
// 		return Message{}, nil, err
// 	}

// 	handler, ok := protocolRegistry[msg.Protocoll]
// 	if !ok {
// 		return msg, nil, fmt.Errorf("no handler for protocol: %s", msg.Protocoll)
// 	}

// 	interpreted, err := handler.InterpretPayload(msg)
// 	return msg, interpreted, err
// }
