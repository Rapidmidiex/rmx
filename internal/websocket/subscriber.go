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
	ic chan *message
	oc chan *message
	// error channel
	errc chan *wserr[CI]
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
		sid: suid.NewUUID(),
		cs:  make(map[suid.UUID]*Conn[CI]),
		// I did make
		ic:             make(chan *message),
		oc:             make(chan *message),
		errc:           make(chan *wserr[CI]),
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
				if err := c.write(p.marshall()); err != nil {
					s.errc <- &wserr[CI]{c, err}
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
		defer c.rwc.Close()

		for {
			// read binary from connection
			b, err := wsutil.ReadClientBinary(c.rwc)
			if err != nil {
				s.errc <- &wserr[CI]{c, err}
				return
			}

			var m message
			m.parse(b)

			switch m.typ {
			case Leave:
				if err := s.disconnect(c); err != nil {
					s.errc <- &wserr[CI]{c, err}
					return
				}
			default:
				s.ic <- &m
			}
		}
	}()
}

// Closes the given Connection and removes it from the Connections list
func (s *Subscriber[SI, CI]) disconnect(c *Conn[CI]) error {
	// close websocket connection
	return c.rwc.Close()
}
