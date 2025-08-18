package p2p

// describe a proxy decorator to protect a generic transport layer
type PeerProxy[D any, T comparable] interface {
	InjectTransport(transport Transport[D, T]) error
}

// type SeedServerProxy struct {
// 	server  Server
// 	encoder Encoder[Message]

// 	mu          sync.Mutex
// 	currentPeer *Peer
// }

// func (s *SeedServerProxy) InjectTransport(server Server) error {
// 	s.server = server

// 	errChan := make(chan error, 1)

// 	go func() {
// 		errChan <- s.server.ListenFor()
// 	}()

// 	go s.handleReceivingRequests()

// 	err := <-errChan

// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

// func (s *SeedServerProxy) handleReceivingRequests() {
// 	eventCh := s.server.GetEventCh()

// 	filteredEventCh := s.filterEventsPolicy(eventCh)

// 	for {
// 		event := <-filteredEventCh

// 		go s.handleHandshake(event)

// 	}
// }

// func (s *SeedServerProxy) handleHandshake(event Event) {

// 	data := event.Data

// 	if data.Name != "Join" {
// 		return
// 	}

// }

// // simple utility function to implement an automatic channel filtering: only currenly managed peer events will be redirect to output channel
// func (s *SeedServerProxy) filterEventsPolicy(eventCh <-chan Event) <-chan Event {
// 	out := make(chan Event)

// 	go func() {
// 		defer close(out)

// 		for ev := range eventCh {
// 			func() {
// 				s.mu.Lock()

// 				if s.currentPeer == nil || s.currentPeer == ev.Source {
// 					s.currentPeer = ev.Source
// 					s.mu.Unlock()
// 					out <- ev
// 				}

// 				s.mu.Unlock()

// 				var peer *Peer = ev.Source

// 				errMsg := Message{
// 					Name: "Error",
// 					Args: map[string]any{
// 						"details": "The seed peer is already managing a connection, try later",
// 					},
// 					Time: 2, //this is an example time, TODO time system
// 				}

// 				byteMsg, _ := s.encoder.Encode(errMsg) //error should not be throwed

// 				ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(1*time.Second)) //generally smaller timeout than default (message can be lost)

// 				defer cancel()

// 				(*peer).Send(ctx, byteMsg) //if send throw an error ignore (consider using a goroutine and adding a mutex to Send)

// 			}()

// 		}
// 	}()

// 	return out
// }
