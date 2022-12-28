package websocket

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/hyphengolang/prelude/types/suid"
	"golang.org/x/sync/errgroup"
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

// Adds a new Subscriber to the list
func (b *Broker[SI, CI]) Subscribe(s *Subscriber[SI, CI]) {
	b.subscribe(s)
}

func (b *Broker[SI, CI]) Unsubscribe(s *Subscriber[SI, CI]) {
	b.unsubscribe(s)
}

func (b *Broker[SI, CI]) Connect(s *Subscriber[SI, CI]) error {
	return b.connect(s)
}

func (b *Broker[SI, CI]) Disconnect(s *Subscriber[SI, CI]) error {
	return b.disconnect(s)
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

func (b *Broker[SI, CI]) subscribe(s *Subscriber[SI, CI]) {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.ss[s.sid] = s
}

func (b *Broker[SI, CI]) unsubscribe(s *Subscriber[SI, CI]) {
	b.lock.Lock()
	defer b.lock.Unlock()
	s.online = false
	close(s.io)
	delete(b.ss, s.sid)
}

func (b *Broker[SI, CI]) connect(s *Subscriber[SI, CI]) error {
	if !s.online {
		s.online = true
	}

	g, _ := errgroup.WithContext(s.Context)

	g.Go(func() error {
		for m := range s.io {
			s.lock.RLock()
			cs := s.cs
			s.lock.RUnlock()

			for _, c := range cs {
				if err := s.write(c, m.marshall()); err != nil {
					return err
				}
			}
		}

		return nil
	})

	// wait for errors
	err := g.Wait()
	if err != nil {
		if err := b.disconnect(s); err != nil {
			return err
		}

		b.unsubscribe(s)
		return err
	}

	return nil
}

func (b *Broker[SI, CI]) disconnect(s *Subscriber[SI, CI]) error {
	for _, c := range s.cs {
		if err := s.disconnect(c); err != nil {
			return err
		}
	}

	return nil
}
