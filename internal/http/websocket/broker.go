package websocket

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"
)

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
	return &Broker[SI, CI]{
		rooms:    make(map[uuid.UUID]*Room[SI, CI]),
		Capacity: cap,
		Context:  ctx,
	}
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

func (b *Broker[SI, CI]) GetSession(sid uuid.UUID) (*Room[SI, CI], error) {
	b.lock.Lock()
	defer b.lock.Unlock()
	s, ok := b.rooms[sid]

	if !ok {
		return nil, errors.New("session not found")
	}

	return s, nil
}

func (b *Broker[SI, CI]) ListSessions() []*Room[SI, CI] {
	b.lock.RLock()
	defer b.lock.RUnlock()

	ss := make([]*Room[SI, CI], 0, len(b.rooms))
	for _, s := range b.rooms {
		ss = append(ss, s)
	}

	return ss
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
