package websocket

import (
	"context"
	"io"
	"log"
	"sync"
	"time"

	"github.com/gobwas/ws/wsutil"
	"github.com/hyphengolang/prelude/types/suid"
)

// Subscriber contains the list of the connections
type Subscriber[SI, CI any] struct {
	// unique id for the Subscriber
	sid  suid.UUID
	lock sync.RWMutex
	// list of Connections
	cs map[suid.UUID]*Conn[CI]
	// Subscriber status
	online bool
	// Input/Output channel for new messages
	ic chan *wsutil.Message
	oc chan *wsutil.Message
	// error channel
	errc chan *wsErr[CI]
	// Maximum Capacity clients allowed
	Capacity uint
	// Maximum message size allowed from peer.
	ReadBufferSize int64
	// Time allowed to read the next pong message from the peer.
	ReadTimeout time.Duration
	// Time allowed to write a message to the peer.
	WriteTimeout time.Duration
	// Info binds its value(like a Jam session) to the subscriber
	Info    *SI
	Context context.Context
}

func NewSubscriber[SI, CI any](
	ctx context.Context,
	cap uint,
	rs int64,
	rt time.Duration,
	wt time.Duration,
	i *SI,
) *Subscriber[SI, CI] {
	s := &Subscriber[SI, CI]{
		sid:            suid.NewUUID(),
		cs:             make(map[suid.UUID]*Conn[CI]),
		ic:             make(chan *wsutil.Message),
		oc:             make(chan *wsutil.Message),
		errc:           make(chan *wsErr[CI]),
		Capacity:       cap,
		ReadBufferSize: rs,
		ReadTimeout:    rt,
		WriteTimeout:   wt,
		Info:           i,
		Context:        ctx,
	}

	s.catch()
	s.listen()

	return s
}

func (s *Subscriber[SI, CI]) NewConn(rwc io.ReadWriteCloser, info *CI) *Conn[CI] {
	return &Conn[CI]{
		sid:  suid.NewUUID(),
		rwc:  rwc,
		Info: info,
	}
}

func (s *Subscriber[SI, CI]) Subscribe(c *Conn[CI]) {
	s.connect(c)
	s.add(c)
}

func (s *Subscriber[SI, CI]) Unsubscribe(c *Conn[CI]) error {
	if err := s.disconnect(c); err != nil {
		return err
	}
	s.remove(c)
	return nil
}

// func (s *Subscriber[SI, CI]) Connect(c *Conn[CI]) error {
// 	return s.connect(c)
// }

// func (s *Subscriber[SI, CI]) Disconnect(c *Conn[CI]) error {
// 	return s.disconnect(c)
// }

func (s *Subscriber[SI, CI]) ListConns() []*Conn[CI] {
	s.lock.RLock()
	defer s.lock.RUnlock()

	conns := make([]*Conn[CI], 0, len(s.cs))
	for _, sub := range s.cs {
		conns = append(conns, sub)
	}

	return conns
}

func (s *Subscriber[SI, CI]) IsFull() bool {
	if s.Capacity == 0 {
		return false
	}

	return len(s.cs) >= int(s.Capacity)
}

func (s *Subscriber[SI, CI]) GetID() suid.UUID {
	return s.sid
}

// listen to the input channel and broadcast messages to clients.
func (s *Subscriber[SI, CI]) listen() {
	go func() {
		for p := range s.ic {
			for _, c := range s.cs {
				if err := wsutil.WriteClientMessage(c.rwc, p.OpCode, p.Payload); err != nil {
					s.errc <- &wsErr[CI]{c, err}
					return
				}
			}
		}
	}()
}

func (s *Subscriber[SI, CI]) catch() {
	go func() {
		for e := range s.errc {
			if err := s.disconnect(e.conn); err != nil {
				log.Println(err)
			}
		}
	}()
}

func (s *Subscriber[SI, CI]) add(c *Conn[CI]) {
	s.lock.Lock()
	defer s.lock.Unlock()

	// add the connection to the list
	s.cs[c.sid] = c
}

func (s *Subscriber[SI, CI]) remove(c *Conn[CI]) {
	s.lock.Lock()
	defer s.lock.Unlock()

	// remove connection from the list
	delete(s.cs, c.sid)
}

// Connects the given Connection to the Subscriber and starts reading from it
func (s *Subscriber[SI, CI]) connect(c *Conn[CI]) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	go func() {
		defer func() {
			if err := s.disconnect(c); err != nil {
				s.errc <- &wsErr[CI]{c, err}
				return
			}
		}()

		for {
			// read binary from connection
			b, op, err := wsutil.ReadClientData(c.rwc)
			if err != nil {
				s.errc <- &wsErr[CI]{c, err}
				return
			}

			if err != nil {
				s.errc <- &wsErr[CI]{c, err}
				return
			}

			s.ic <- &wsutil.Message{OpCode: op, Payload: b}
		}
	}()
}

// Closes the given Connection and removes it from the Connections list
func (s *Subscriber[SI, CI]) disconnect(c *Conn[CI]) error {
	// close websocket connection
	return c.rwc.Close()
}
