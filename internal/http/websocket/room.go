package websocket

import (
	"context"
	"io"
	"log"
	"sync"
	"time"

	"github.com/gobwas/ws/wsutil"
	"github.com/google/uuid"
	"github.com/hyphengolang/prelude/types/suid"
)

// Room contains the list of the connections
type (
	Room[RoomType, ConnType any] struct {
		// unique id for the Subscriber
		sid  uuid.UUID
		lock sync.RWMutex
		// list of Connections
		cs map[uuid.UUID]*Conn[ConnType]
		// error channel
		errc chan *wsErr[ConnType]
		// Maximum Capacity clients allowed
		Capacity uint
		// Maximum message size allowed from peer.
		ReadBufferSize int64
		// Time allowed to read the next pong message from the peer.
		ReadTimeout time.Duration
		// Time allowed to write a message to the peer.
		WriteTimeout time.Duration
		Context      context.Context
	}

	NewRoomArgs struct {
		Context        context.Context
		Capacity       uint
		ReadBufferSize int64
		ReadTimeout    time.Duration
		WriteTimeout   time.Duration
		JamID          uuid.UUID
	}
)

func NewRoom[RoomType, ConnType any](args NewRoomArgs) *Room[RoomType, ConnType] {
	s := &Room[RoomType, ConnType]{
		sid:            args.JamID,
		cs:             make(map[uuid.UUID]*Conn[ConnType]),
		errc:           make(chan *wsErr[ConnType]),
		Capacity:       args.Capacity,
		ReadBufferSize: args.ReadBufferSize,
		ReadTimeout:    args.ReadTimeout,
		WriteTimeout:   args.WriteTimeout,
		Context:        args.Context,
	}

	s.catch()
	return s
}

func (s *Room[RoomType, ConnType]) NewConn(rwc io.ReadWriteCloser, info *ConnType) *Conn[ConnType] {
	return &Conn[ConnType]{
		sid:  suid.NewUUID(),
		rwc:  rwc,
		Info: info,
	}
}

func (s *Room[SI, CI]) Subscribe(c *Conn[CI]) {
	s.connect(c)
	s.add(c)
}

func (s *Room[SI, CI]) Unsubscribe(c *Conn[CI]) error {
	if err := s.disconnect(c); err != nil {
		return err
	}
	s.remove(c)
	return nil
}

func (s *Room[SI, CI]) IsFull() bool {
	if s.Capacity == 0 {
		return false
	}

	return len(s.cs) >= int(s.Capacity)
}

func (s *Room[SI, CI]) ID() uuid.UUID {
	return s.sid
}

// listen to the input channel and broadcast messages to clients.
func (s *Room[SI, CI]) broadcast(m *wsutil.Message) {
	for _, c := range s.cs {
		if err := c.write(m); err != nil {
			s.errc <- &wsErr[CI]{c, err}
			return
		}
	}
}

func (s *Room[SI, CI]) catch() {
	go func() {
		for e := range s.errc {
			if err := s.disconnect(e.conn); err != nil {
				log.Println(err)
			}
		}
	}()
}

func (s *Room[SI, CI]) add(c *Conn[CI]) {
	s.lock.Lock()
	defer s.lock.Unlock()

	// add the connection to the list
	s.cs[c.sid.UUID] = c
}

func (s *Room[SI, CI]) remove(c *Conn[CI]) {
	s.lock.Lock()
	defer s.lock.Unlock()

	// remove connection from the list
	delete(s.cs, c.sid.UUID)
}

// Connects the given Connection to the Subscriber and starts reading from it
func (s *Room[SI, CI]) connect(c *Conn[CI]) {
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

			m := &wsutil.Message{OpCode: op, Payload: b}

			s.broadcast(m)
		}
	}()
}

// Closes the given Connection and removes it from the Connections list
func (s *Room[SI, CI]) disconnect(c *Conn[CI]) error {
	// close websocket connection
	return c.rwc.Close()
}
