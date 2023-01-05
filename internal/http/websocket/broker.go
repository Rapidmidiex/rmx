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

// Adds a new Subscriber to the list
func (b *Broker[SI, CI]) Subscribe(s *Session[SI, CI]) {
	b.connect(s)
	b.add(s)
}

func (b *Broker[SI, CI]) Unsubscribe(s *Session[SI, CI]) error {
	if err := b.disconnect(s); err != nil {
		return err
	}
	b.remove(s)
	return nil
}

func (b *Broker[SI, CI]) Connect(s *Session[SI, CI]) {
	b.connect(s)
}

func (b *Broker[SI, CI]) Disconnect(s *Session[SI, CI]) error {
	return b.disconnect(s)
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

	ss := make([]*Session[SI, CI], 0, len(b.ss))
	for _, s := range b.ss {
		ss = append(ss, s)
	}

	return ss
}

func (b *Broker[SI, CI]) add(s *Session[SI, CI]) {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.ss[s.sid] = s
}

func (b *Broker[SI, CI]) remove(s *Session[SI, CI]) {
	b.lock.Lock()
	defer b.lock.Unlock()
	close(s.ic)
	close(s.oc)
	close(s.errc)
	delete(b.ss, s.sid)
}

func (b *Broker[SI, CI]) connect(s *Session[SI, CI]) {
	if !s.online {
		s.online = true
	}

	go func() {
		for m := range s.ic {
			// s.lock.RLock()
			cs := s.cs
			// s.lock.RUnlock()

			for _, c := range cs {
				if err := c.write(m.marshall()); err != nil {
					s.errc <- &wsErr[CI]{c, err}
					return
				}
			}
		}
	}()
}

func (b *Broker[SI, CI]) disconnect(s *Session[SI, CI]) error {
	s.online = false
	for _, c := range s.cs {
		if err := s.disconnect(c); err != nil {
			return err
		}
	}

	return nil
}
