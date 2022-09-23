package websocket

import (
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/rog-golang-buddies/rapidmidiex/internal/suid"

	// https://stackoverflow.com/questions/21362950/getting-a-slice-of-keys-from-a-map

	"golang.org/x/exp/maps"
)

type Pool struct {
	mu sync.RWMutex

	ID      suid.UUID
	MaxConn int

	cs   map[suid.UUID]*Conn
	msgs chan any
}

func DefaultPool() *Pool {
	p := &Pool{
		ID:      suid.NewUUID(),
		MaxConn: 4,
		cs:      make(map[suid.UUID]*Conn),
		msgs:    make(chan any),
	}

	go func() {
		defer p.Close()

		for msg := range p.msgs {
			for _, c := range p.cs {
				c.WriteJSON(msg)
			}
		}

		// ?why does this pattern not work
		// for _, c := range p.cs {	c.WriteJSON(<-p.msgs) }
	}()

	return p
}

func (p *Pool) Size() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	return len(p.cs)
}

func (p *Pool) Keys() []suid.UUID {
	p.mu.Lock()
	defer p.mu.Unlock()

	return maps.Keys(p.cs)
}

func (p *Pool) NewConn(w http.ResponseWriter, r *http.Request, u *websocket.Upgrader) (*Conn, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	rwc, err := u.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}

	c := &Conn{suid.NewUUID(), rwc, p}

	p.cs[c.ID] = c

	return c, nil
}

func (p *Pool) Delete(uid suid.UUID) {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.cs, uid)
}

func (p *Pool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, c := range p.cs {
		if err := c.Close(); err != nil {
			return err
		}
	}

	return nil
}

/*
Experimental: Pub/Sub pattern

func (p *Pool) AddEventListener(eventTyp string, callback func(event any)) {
	if eventTyp already exists then panic
	if callback is nil then panic

	type entry struct { eventTyp string; callback func(event any); }
	add entry to to map[eventType]entry
}

func (p *Pool) Listen() {

}
*/
