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
	ss map[suid.UUID]*Subscriber[SI, CI]

	// Maximum Capacity Subscribers allowed
	Capacity uint
	Context  context.Context
}

func NewBroker[SI, CI any](cap uint, ctx context.Context) *Broker[SI, CI] {
	return &Broker[SI, CI]{
		ss:       make(map[suid.UUID]*Subscriber[SI, CI]),
		Capacity: cap,
		Context:  ctx,
	}
}

// Adds a new Subscriber to the list
func (b *Broker[SI, CI]) Subscribe(s *Subscriber[SI, CI]) error {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.ss[s.sid] = s

	return nil
}

func (b *Broker[SI, CI]) GetSubscriber(sid suid.UUID) (*Subscriber[SI, CI], error) {
	b.lock.Lock()
	defer b.lock.Unlock()
	s, ok := b.ss[sid]

	if !ok {
		return nil, errors.New("Subscriber not found")
	}

	return s, nil
}
