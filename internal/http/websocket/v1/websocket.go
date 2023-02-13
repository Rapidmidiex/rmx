package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/google/uuid"
	"github.com/hyphengolang/prelude/types/suid"
	"github.com/rapidmidiex/rmx/internal/msg"
)

type wsErr[CI any] struct {
	conn *Conn[CI]
	msg  error
}

func (e *wsErr[CI]) Error() string {
	return e.msg.Error()
}

// A Web-Socket Connection
type Conn[CI any] struct {
	sid  suid.UUID
	rwc  io.ReadWriteCloser
	lock sync.RWMutex

	Info *CI
}

// Writes raw bytes to the Connection
func (c *Conn[CI]) write(m *wsutil.Message) error {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return wsutil.WriteServerMessage(c.rwc, m.OpCode, m.Payload)
}

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

// Subscribe sets up a read loop on the Connection and broadcasts all messages to the other Connections in the Room.
func (r *Room[SI, CI]) Subscribe(c *Conn[CI]) error {
	r.connect(c)
	r.add(c)

	// TODO Get user from DB or gen anon user
	conMsg := msg.ConnectMsg{
		UserID:   c.sid.UUID,
		UserName: gofakeit.Username(),
	}
	envelope := msg.Envelope{
		Typ:    msg.CONNECT,
		UserID: conMsg.UserID,
	}
	err := envelope.SetPayload(conMsg)
	if err != nil {
		return fmt.Errorf("marshall payload: %w", err)
	}
	// Write connect message back to client
	payload, err := json.Marshal(&envelope)
	if err != nil {
		return fmt.Errorf("marshall envelope: %w", err)
	}

	return c.write(&wsutil.Message{
		OpCode:  ws.OpText,
		Payload: payload,
	})
}

// Unsubscribe disconnects the given Connection and removes it from the Connections list.
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

// Broadcast listens to the input channel and broadcasts messages to clients.
func (r *Room[SI, CI]) broadcast(m *wsutil.Message) {
	for _, c := range r.cs {
		if err := c.write(m); err != nil && err != io.EOF {
			r.errc <- &wsErr[CI]{c, fmt.Errorf("broadcast: write: %w", err)}
		}
	}
}

func (r *Room[SI, CI]) catch() {
	go func() {
		for e := range r.errc {
			log.Printf("room err: %s\nunsubscribing..", e)
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
		for {
			// read binary from connection
			b, op, err := wsutil.ReadClientData(c.rwc)
			if err != nil && err != io.EOF {
				// TODO: Handle peer CONNRESET.
				// We don't want to close the connection for everyone if one client goes down
				r.errc <- &wsErr[CI]{c, fmt.Errorf("connect: readClientData: %w", err)}
				return
			}

			// Only broadcast Text messages
			// Not client closes,ping/pong, etc
			// Maybe use ws.isData()
			if op.IsData() {
				log.Printf("OpCode: %v\nisData: %v\n\n", OpCodeToString(op), op.IsData())
				m := &wsutil.Message{OpCode: op, Payload: b}

				r.broadcast(m)
				continue
			}

			// TODO: Handle other op codes
			log.Printf("unhandled op code: %v\nisData: %v\n\n", OpCodeToString(op), op.IsData())

		}
	}()
}

// Disconnect closes the given connection.
func (r *Room[SI, CI]) disconnect(c *Conn[CI]) error {
	// check if connection exists
	_, ok := r.cs[c.sid.UUID]

	// close websocket connection
	if !ok {
		return ErrConnNotFound
	}

	return c.rwc.Close()
}

// Broker contains the list of the Rooms
type (
	Broker[SI, CI any] struct {
		lock sync.RWMutex
		// List of rooms.
		rooms map[uuid.UUID]*Room[SI, CI]

		// Maximum total rooms allowed.
		Capacity uint
		Context  context.Context
	}
)

var ErrRoomNotFound = errors.New("room not found")

func NewBroker[SI, CI any](cap uint, ctx context.Context) *Broker[SI, CI] {
	b := &Broker[SI, CI]{
		rooms:    make(map[uuid.UUID]*Room[SI, CI]),
		Capacity: cap,
		Context:  ctx,
	}

	return b
}

// Adds a new Subscriber to the list
func (b *Broker[SI, CI]) Subscribe(s *Room[SI, CI]) {
	b.add(s)
}

func (b *Broker[SI, CI]) Unsubscribe(s *Room[SI, CI]) error {
	if err := b.close(s); err != nil {
		return err
	}
	b.remove(s)
	return nil
}

// GetRoom retrieves a room by Jam ID. If none found a new room is created.
func (b *Broker[SI, CI]) GetRoom(sid uuid.UUID) (*Room[SI, CI], error) {
	b.lock.Lock()
	defer b.lock.Unlock()
	s, ok := b.rooms[sid]

	if !ok {
		return s, ErrRoomNotFound
	}

	return s, nil
}

// ConnCount returns the current number of active connections.
func (b *Broker[SI, CI]) ConnCount(sid uuid.UUID) int {
	b.lock.Lock()
	defer b.lock.Unlock()
	s, ok := b.rooms[sid]

	if !ok {
		// If no room found, just return 0
		return 0
	}

	return len(s.cs)
}

func (b *Broker[SI, CI]) add(s *Room[SI, CI]) {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.rooms[s.sid] = s
}

func (b *Broker[SI, CI]) remove(s *Room[SI, CI]) {
	b.lock.Lock()
	defer b.lock.Unlock()
	close(s.errc)
	delete(b.rooms, s.sid)
}

func (b *Broker[SI, CI]) close(s *Room[SI, CI]) error {
	for _, c := range s.cs {
		if err := s.disconnect(c); err != nil {
			return err
		}
	}

	return nil
}

type Reader interface {
	ReadText() (string, error)
	ReadJSON() (interface{}, error)
}

type Writer interface {
	WriteText(s string)
	WriteJSON(i any) error
}

func OpCodeToString(o ws.OpCode) string {
	switch o {
	case ws.OpContinuation:
		return "OpContinuation"
	case ws.OpText:
		return "OpText"
	case ws.OpBinary:
		return "OpBinary"
	case ws.OpClose:
		return "OpClose"
	case ws.OpPing:
		return "OpPing"
	case ws.OpPong:
		return "OpPong"
	}
	return "Unknown"
}
