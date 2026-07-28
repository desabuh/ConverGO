package p2p

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
)

const (
	PING_RETRIEVE   = "PING_RETRIEVE_TOPIC"
	PING_JOIN       = "PING_JOIN_TOPIC"
	PUSH            = "PUSH_TOPIC"
	PEER_CONNECTION = "PEER_CONNECTION_TOPIC"
	PEER_EXIT       = "PEER_EXIT_TOPIC"
	CLUSTER_JOIN    = "CLUSTER_JOIN_TOPIC"
)

type SessionRole int

const (
	SessionInitiator SessionRole = iota
	SessionReceiver
)

type SessionId string
type SeqNo int32

const NEW_MSG_SEQUENCE SeqNo = 0
const INITIATED_MSG_SEQUENCE SeqNo = 1

type MsgType string
type TopicName string

type SessionFields struct {
	TopicName TopicName
	SessionId SessionId
	SeqNo     SeqNo
}

// Rappresent an open session over a communication pipe that works with messages M trasmitted between T object
// It also provide a way to wait on a session and close it
type Session[M comparable, T comparable] interface {
	Send(ctx context.Context, data M, target T) error
	Broadcast(ctx context.Context, data M) error
	WaitOn(ctx context.Context) (M, error)
	GetTopic() TopicName
	Close()
}

type PeerMessageSession[T comparable] struct {
	Id    SessionId
	topic TopicName

	transport CommunicationPipe[PeerMessage, T]
	Recv      chan PeerMessage

	mu           sync.Mutex // mu protects CurrentSeqNo and isClosed.
	CurrentSeqNo SeqNo
	isClosed     bool

	onClose func(SessionId)
}

func (s *PeerMessageSession[T]) GetTopic() TopicName {
	return s.topic
}

func (s *PeerMessageSession[T]) Send(ctx context.Context, data PeerMessage, target T) error {
	data, err := s.setupSendingMsg(data)

	if err != nil {
		return err
	}

	return s.transport.Send(ctx, data, target)
}

func (s *PeerMessageSession[T]) Broadcast(ctx context.Context, data PeerMessage) error {
	data, err := s.setupSendingMsg(data)

	if err != nil {
		return err
	}

	return s.transport.Broadcast(ctx, data)
}

func (s *PeerMessageSession[T]) setupSendingMsg(data PeerMessage) (PeerMessage, error) {
	s.mu.Lock()
	if s.isClosed {
		s.mu.Unlock()
		return PeerMessage{}, fmt.Errorf("session was already closed")
	}

	data.Key.SeqNo = s.CurrentSeqNo
	s.CurrentSeqNo++
	data.Key.SessionId = s.Id
	data.Key.TopicName = s.topic
	s.mu.Unlock()

	return data, nil
}

func (s *PeerMessageSession[T]) WaitOn(ctx context.Context) (PeerMessage, error) {
	s.mu.Lock()
	if s.isClosed {
		s.mu.Unlock()
		return PeerMessage{}, fmt.Errorf("session was already closed")
	}
	recvCh := s.Recv
	s.mu.Unlock()

	select {
	case msg, ok := <-recvCh:
		if !ok {
			return PeerMessage{}, fmt.Errorf("session was already closed")
		}

		s.mu.Lock()
		defer s.mu.Unlock()

		if s.isClosed {
			return PeerMessage{}, fmt.Errorf("session was already closed")
		}

		if msg.Key.SeqNo != s.CurrentSeqNo {
			return PeerMessage{}, fmt.Errorf(
				"message for topic %s was received out of order, proceed to drop it",
				s.topic,
			)
		}

		return msg, nil

	case <-ctx.Done():
		return PeerMessage{}, ctx.Err()
	}
}

func (s *PeerMessageSession[T]) Close() {
	s.mu.Lock()
	if s.isClosed {
		s.mu.Unlock()
		return
	}
	s.isClosed = true
	close(s.Recv)
	onClose := s.onClose
	id := s.Id
	s.mu.Unlock()

	if onClose != nil {
		onClose(id)
	}
}

type TopicHandlerFunc[T comparable, M comparable] func(ctx context.Context, session Session[M, T], msg M)

// a general message broker to provide some delivery semantic regarding messages M received and transmitted towards a target T
type MessageBroker[M comparable, T comparable] interface {
	RegisterHandler(topic TopicName, handler TopicHandlerFunc[T, M]) MessageBroker[M, T]
	StartWaiting(ctx context.Context) error
	CreateNewSession(topic TopicName) Session[M, T]
	Stop()
}

type PeerMessageBroker[T comparable] struct {
	transport         CommunicationPipe[PeerMessage, T]
	sessionBufferSize int

	mu        sync.Mutex // mu protects handlers, consumers, stopped, runCtx, and cancel.
	handlers  map[TopicName][]TopicHandlerFunc[T, PeerMessage]
	consumers map[SessionId]*PeerMessageSession[T]
	stopped   bool
	runCtx    context.Context
	cancel    context.CancelFunc
}

func NewPeerMessageBroker[T comparable](
	transport CommunicationPipe[PeerMessage, T],
	sessionBufferSize int,
) *PeerMessageBroker[T] {
	return &PeerMessageBroker[T]{
		transport:         transport,
		sessionBufferSize: sessionBufferSize,
		handlers:          make(map[TopicName][]TopicHandlerFunc[T, PeerMessage]),
		consumers:         make(map[SessionId]*PeerMessageSession[T]),
	}
}

func (mb *PeerMessageBroker[T]) RegisterHandler(topic TopicName, handler TopicHandlerFunc[T, PeerMessage]) MessageBroker[PeerMessage, T] {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	if mb.stopped {
		return mb
	}

	mb.handlers[topic] = append(mb.handlers[topic], handler)
	return mb
}

func (mb *PeerMessageBroker[T]) StartWaiting(ctx context.Context) error {
	mb.mu.Lock()
	if mb.stopped {
		mb.mu.Unlock()
		return fmt.Errorf("broker already stopped")
	}
	if mb.runCtx != nil {
		mb.mu.Unlock()
		return fmt.Errorf("broker already started")
	}

	runCtx, cancel := context.WithCancel(ctx)
	mb.runCtx = runCtx
	mb.cancel = cancel
	mb.mu.Unlock()

	defer func() {
		mb.mu.Lock()
		if mb.cancel != nil {
			mb.cancel()
		}
		mb.runCtx = nil
		mb.cancel = nil
		mb.mu.Unlock()
	}()

	recvCh := mb.transport.GetReceiveCh()

	for {
		select {
		case <-runCtx.Done():
			return runCtx.Err()

		case msg, ok := <-recvCh:
			if !ok {
				return fmt.Errorf("transport receive channel closed")
			}
			mb.dispatchMessage(runCtx, msg)
		}
	}
}

func (mb *PeerMessageBroker[T]) dispatchMessage(ctx context.Context, msg PeerMessage) {
	mb.mu.Lock()
	if mb.stopped {
		mb.mu.Unlock()
		return
	}

	topic := msg.Key.TopicName

	handlers := append([]TopicHandlerFunc[T, PeerMessage](nil), mb.handlers[topic]...)

	//if message have starting SEQNO create a new session and execute it through each handler (if no handler are registered nothing will happen)
	if msg.Key.SeqNo == NEW_MSG_SEQUENCE {
		sessions := make([]Session[PeerMessage, T], 0, len(handlers))
		for range handlers {
			sessions = append(sessions, mb.createSessionLocked(topic, SessionReceiver, msg.Key.SessionId))
		}
		mb.mu.Unlock()

		for i, handler := range handlers {
			session := sessions[i]
			go func(h TopicHandlerFunc[T, PeerMessage], s Session[PeerMessage, T], firstMsg PeerMessage) {
				defer s.Close()
				h(ctx, s, firstMsg)
			}(handler, session, msg)
		}
		return
	}

	session := mb.consumers[msg.Key.SessionId]

	mb.mu.Unlock()

	if session == nil {
		return
	}

	select {
	case session.Recv <- msg:
	default:
	}
}

func (mb *PeerMessageBroker[T]) createSessionLocked(topic TopicName, role SessionRole, id SessionId) Session[PeerMessage, T] {

	var startingSeqNo SeqNo
	if role == SessionInitiator {
		startingSeqNo = NEW_MSG_SEQUENCE
	} else {
		startingSeqNo = INITIATED_MSG_SEQUENCE
	}

	session := &PeerMessageSession[T]{
		Id:           id,
		topic:        topic,
		transport:    mb.transport,
		Recv:         make(chan PeerMessage, mb.sessionBufferSize),
		CurrentSeqNo: startingSeqNo,
		onClose:      mb.removeSession,
	}

	mb.consumers[id] = session
	return session
}

func (mb *PeerMessageBroker[T]) CreateNewSession(topic TopicName) Session[PeerMessage, T] {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	if mb.stopped {
		return nil
	}

	newSessionId := uuid.New().String()

	return mb.createSessionLocked(topic, SessionInitiator, SessionId(newSessionId))
}

func (mb *PeerMessageBroker[T]) removeSession(id SessionId) {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	delete(mb.consumers, id)
}

func (mb *PeerMessageBroker[T]) Stop() {
	mb.mu.Lock()
	if mb.stopped {
		mb.mu.Unlock()
		return
	}

	mb.stopped = true
	cancel := mb.cancel

	sessions := make([]*PeerMessageSession[T], 0, len(mb.consumers))
	for _, s := range mb.consumers {
		sessions = append(sessions, s)
	}
	mb.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	for _, s := range sessions {
		s.Close()
	}
}
