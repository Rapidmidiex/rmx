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

func (s *Subscriber[SI, CI]) NewConn(rwc io.ReadWriteCloser, info *CI) *Conn[CI] {
	return &Conn[CI]{
		sid:  suid.NewUUID(),
		rwc:  rwc,
		Info: info,
	}
}

func (s *Subscriber[SI, CI]) Subscribe(c *Conn[CI]) {
	s.subscribe(c)
}

func (s *Subscriber[SI, CI]) Unsubscribe(c *Conn[CI]) {
	s.unsubscribe(c)
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

func (s *Subscriber[SI, CI]) GetID() suid.UUID {
	return s.sid
}

func (s *Subscriber[SI, CI]) subscribe(c *Conn[CI]) {
	s.lock.Lock()
	defer s.lock.Unlock()

	// add the connection to the list
	s.cs[c.sid] = c
}

func (s *Subscriber[SI, CI]) unsubscribe(c *Conn[CI]) {
	s.lock.Lock()
	defer s.lock.Unlock()

	// remove connection from the list
	delete(s.cs, c.sid)
}

// Connects the given Connection to the Subscriber and adds it to the list of its Connections
func (s *Subscriber[SI, CI]) connect(c *Conn[CI]) error {
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

	return nil
}

// Closes the given Connection and removes it from the Connections list
func (s *Subscriber[SI, CI]) disconnect(c *Conn[CI]) error {
	// close websocket connection
	return c.rwc.Close()
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

	var m message
	m.parse(b)

	switch m.typ {
	case Leave:
		return s.disconnect(c)
	default:
		s.io <- &m
	}

	return nil
}

// Writes raw bytes to the Connection
func (s *Subscriber[SI, CI]) write(c *Conn[CI], b []byte) error {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return wsutil.WriteServerBinary(c.rwc, b)
}
