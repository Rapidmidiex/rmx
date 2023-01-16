package websocket

import (
	"context"
	"errors"
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

var ErrConnNotFound = errors.New("connection not found")

func NewRoom[RoomType, ConnType any](args NewRoomArgs) *Room[RoomType, ConnType] {
	r := &Room[RoomType, ConnType]{
		sid:            args.JamID,
		cs:             make(map[uuid.UUID]*Conn[ConnType]),
		errc:           make(chan *wsErr[ConnType]),
		Capacity:       args.Capacity,
		ReadBufferSize: args.ReadBufferSize,
		ReadTimeout:    args.ReadTimeout,
		WriteTimeout:   args.WriteTimeout,
		Context:        args.Context,
	}

	r.catch()
	return r
}

func (r *Room[RoomType, ConnType]) NewConn(rwc io.ReadWriteCloser, info *ConnType) *Conn[ConnType] {
	return &Conn[ConnType]{
		sid:  suid.NewUUID(),
		rwc:  rwc,
		Info: info,
	}
}

func (r *Room[SI, CI]) Subscribe(c *Conn[CI]) {
	r.connect(c)
	r.add(c)
}

func (r *Room[SI, CI]) Unsubscribe(c *Conn[CI]) error {
	if err := r.disconnect(c); err != nil {
		return err
	}
	r.remove(c)
	return nil
}

func (r *Room[SI, CI]) IsFull() bool {
	if r.Capacity == 0 {
		return false
	}

	return len(r.cs) >= int(r.Capacity)
}

func (r *Room[SI, CI]) ID() uuid.UUID {
	return r.sid
}

// listen to the input channel and broadcast messages to clients.
func (r *Room[SI, CI]) broadcast(m *wsutil.Message) {
	for _, c := range r.cs {
		if err := c.write(m); err != nil {
			log.Println("here2")
			r.errc <- &wsErr[CI]{c, err}
			return
		}
	}
}

func (r *Room[SI, CI]) catch() {
	go func() {
		for e := range r.errc {
			if err := r.Unsubscribe(e.conn); err != nil {
				log.Println(err)
			}
		}
	}()
}

func (r *Room[SI, CI]) add(c *Conn[CI]) {
	r.lock.Lock()
	defer r.lock.Unlock()

	// add the connection to the list
	r.cs[c.sid.UUID] = c
}

func (r *Room[SI, CI]) remove(c *Conn[CI]) {
	r.lock.Lock()
	defer r.lock.Unlock()

	// remove connection from the list
	delete(r.cs, c.sid.UUID)
}

// Connects the given Connection to the Subscriber and starts reading from it
func (r *Room[SI, CI]) connect(c *Conn[CI]) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	go func() {
		defer func() {
			if err := r.Unsubscribe(c); err != nil {
				log.Println("here")
				r.errc <- &wsErr[CI]{c, err}
				return
			}
		}()

		for {
			// read binary from connection
			b, op, err := wsutil.ReadClientData(c.rwc)
			if err != nil {
				log.Println("here1")
				r.errc <- &wsErr[CI]{c, err}
				return
			}

			m := &wsutil.Message{OpCode: op, Payload: b}

			r.broadcast(m)
		}
	}()
}

// Closes the given Connection and removes it from the Connections list
func (r *Room[SI, CI]) disconnect(c *Conn[CI]) error {
	// check if connection exists
	_, ok := r.cs[c.sid.UUID]

	// close websocket connection
	if !ok {
		return ErrConnNotFound
	}

	return c.rwc.Close()
}
