package p2p

import "github.com/google/uuid"

// type Event struct {
// 	Source *Peer
// 	Data   Message
// }

type EventType string

const (
	EventTypeHandshake EventType = "HANDSHAKE"
	EventTypeFile      EventType = "FILE"
	EventTypeError     EventType = "ERROR"
	EventTypeRequest   EventType = "REQUEST"
	EventTypeResponse  EventType = "RESPONSE"
)

type Command string

const (
	CommandStartHandshake Command = "START_HANDSHAKE"
	CommandAckHandshake   Command = "ACK_HANDSHAKE"
	CommandSendFile       Command = "SEND_FILE"
	CommandCancelFile     Command = "CANCEL_FILE"
	CommandRequestData    Command = "REQUEST_DATA"
	CommandResponseData   Command = "RESPONSE_DATA"
	CommandError          Command = "ERROR"
)

type Event struct {
	EventID        uuid.UUID
	TimeStamp      int64
	EventType      EventType
	RelatedEventID uuid.UUID
	SequenceNum    int
	EndOfEvent     bool
	Command        Command
	Args           map[string]interface{}

	Payload string
}

func NewSingleMessageEvent(payload string, typ EventType, command Command, args ...map[string]interface{}) Event {
	eventID := uuid.New()
	mergedArgs := make(map[string]interface{})
	for _, arg := range args {
		for key, value := range arg {
			mergedArgs[key] = value
		}
	}
	return Event{
		EventID:        eventID,
		RelatedEventID: eventID,
		SequenceNum:    1,
		EndOfEvent:     true,
		Command:        command,
		Payload:        payload,
		Args:           mergedArgs,
	}
}
