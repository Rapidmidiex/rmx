package websocket

import (
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	// https://stackoverflow.com/questions/21362950/getting-a-slice-of-keys-from-a-map

	"golang.org/x/exp/maps"
)

type Pool struct {
	mu sync.Mutex

	ID      uuid.UUID
	MaxConn int

	cs   map[uuid.UUID]*Conn
	msgs chan any
}

func DefaultPool() *Pool {
	p := &Pool{
		ID:      uuid.New(),
		MaxConn: 4,
		cs:      make(map[uuid.UUID]*Conn),
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
		// for _, c := range p.cs {
		// 	c.WriteJSON(<-p.msgs)
		// }
	}()

	return p
}

func (p *Pool) Size() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	return len(p.cs)
}

func (p *Pool) Keys() []uuid.UUID {
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

	c := &Conn{uuid.New(), rwc, p}

	p.cs[c.ID] = c

	return c, nil
}

func (p *Pool) Delete(uid uuid.UUID) {
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
