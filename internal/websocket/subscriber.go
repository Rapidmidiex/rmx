package websocket

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/gobwas/ws/wsutil"
	"github.com/hyphengolang/prelude/types/suid"
	"golang.org/x/sync/errgroup"
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
	io chan *message
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

func (b *Broker[SI, CI]) NewSubscriber(
	cap uint,
	rs int64,
	rt time.Duration,
	wt time.Duration,
	i *SI,
) *Subscriber[SI, CI] {
	return &Subscriber[SI, CI]{
		sid:            suid.NewUUID(),
		cs:             make(map[suid.UUID]*Conn[CI]),
		io:             make(chan *message),
		Capacity:       cap,
		ReadBufferSize: rs,
		ReadTimeout:    rt,
		WriteTimeout:   wt,
		Info:           i,
		Context:        b.Context,
	}
}

func (s *Subscriber[SI, CI]) Listen() error {
	return s.listen()
}

func (s *Subscriber[SI, CI]) Connect(c *Conn[CI]) error {
	return s.connect(c)
}

func (s *Subscriber[SI, CI]) Disconnect(c *Conn[CI]) error {
	return s.disconnect(c)
}

func (s *Subscriber[SI, CI]) IsFull() bool {
	if s.Capacity == 0 {
		return false
	}

	return len(s.cs) >= int(s.Capacity)
}

func (s *Subscriber[SI, CI]) NewConn(rwc io.ReadWriteCloser, info *CI) *Conn[CI] {
	return &Conn[CI]{
		sid:  suid.NewUUID(),
		rwc:  rwc,
		Info: info,
	}
}

func (s *Subscriber[SI, CI]) GetID() suid.UUID {
	return s.sid
}

// Starts listening on the io channel which Connections send their messages to
func (s *Subscriber[SI, CI]) listen() error {
	s.online = true

	for m := range s.io {
		s.lock.RLock()
		cs := s.cs
		s.lock.Unlock()

		for _, c := range cs {
			if err := s.write(c, m.marshall()); err != nil {
				return err
			}
		}
	}

	return nil
}

// Connects the given Connection to the Subscriber and adds it to the list of its Connections
func (s *Subscriber[SI, CI]) connect(c *Conn[CI]) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	// check if the Subscriber is listening
	if s.online {
		// add the connection to the list
		s.cs[c.sid] = c

		// create an error group to catch goroutine errors
		g, _ := errgroup.WithContext(s.Context)
		g.Go(func() error {
			return s.read(c)
		})

		// wait for errors
		err := g.Wait()
		if err != nil {
			if err := s.disconnect(c); err != nil {
				return err
			}

			return err
		}
	}

	return nil
}

// Closes the given Connection and removes it from the Connections list
func (s *Subscriber[SI, CI]) disconnect(c *Conn[CI]) error {
	// close websocket connection
	if err := c.rwc.Close(); err != nil {
		return err
	}

	// remove connection from the list
	delete(s.cs, c.sid)
	return nil
}

// Starts reading from the given Connection
func (s *Subscriber[SI, CI]) read(c *Conn[CI]) error {
	c.lock.RLock()
	defer c.lock.RUnlock()

	// read binary from connection
	b, err := wsutil.ReadClientBinary(c.rwc)
	if err != nil {
		return err
	}

	var m *message
	m.parse(b)

	switch m.typ {
	case Leave:
		return s.disconnect(c)
	default:
		s.io <- m
	}

	return nil
}

// Writes raw bytes to the Connection
func (s *Subscriber[SI, CI]) write(c *Conn[CI], b []byte) error {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return wsutil.WriteServerBinary(c.rwc, b)
}
