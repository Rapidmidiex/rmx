package websocket

import (
	"context"
	"errors"
	"sync"

	"github.com/hyphengolang/prelude/types/suid"
)

// Broker contains the list of the Subscribers
type Broker[SI, CI any] struct {
	lock sync.RWMutex
	// list of Subscribers
	ss map[suid.UUID]*Session[SI, CI]

	// Maximum Capacity Subscribers allowed
	Capacity uint
	Context  context.Context
}

func NewBroker[SI, CI any](cap uint, ctx context.Context) *Broker[SI, CI] {
	return &Broker[SI, CI]{
		ss:       make(map[suid.UUID]*Session[SI, CI]),
		Capacity: cap,
		Context:  ctx,
	}
}

// Adds a new Session to the list
func (b *Broker[SI, CI]) Subscribe(s *Session[SI, CI]) {
	b.add(s)
}

func (b *Broker[SI, CI]) Unsubscribe(s *Session[SI, CI]) error {
	if err := b.close(s); err != nil {
		return err
	}
	b.remove(s)
	return nil
}

func (b *Broker[SI, CI]) GetSession(sid suid.UUID) (*Session[SI, CI], error) {
	b.lock.Lock()
	defer b.lock.Unlock()
	s, ok := b.ss[sid]

	if !ok {
		return nil, errors.New("session not found")
	}

	return s, nil
}

func (b *Broker[SI, CI]) ListSessions() []*Session[SI, CI] {
	b.lock.RLock()
	defer b.lock.RUnlock()

	subs := make([]*Session[SI, CI], 0, len(b.ss))
	for _, sub := range b.ss {
		subs = append(subs, sub)
	}

	return subs
}

func (b *Broker[SI, CI]) add(s *Session[SI, CI]) {
	b.lock.Lock()
	defer b.lock.Unlock()
	s.online = true
	b.ss[s.sid] = s
}

func (b *Broker[SI, CI]) remove(s *Session[SI, CI]) {
	b.lock.Lock()
	defer b.lock.Unlock()
	close(s.errc)
	delete(b.ss, s.sid)
}

func (b *Broker[SI, CI]) close(s *Session[SI, CI]) error {
	s.online = false
	for _, c := range s.cs {
		if err := s.disconnect(c); err != nil {
			return err
		}
	}

	return nil
}
