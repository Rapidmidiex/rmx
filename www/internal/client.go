package websocket

import (
	"sync"

	rmx "github.com/rog-golang-buddies/rapidmidiex/internal"
	"github.com/rog-golang-buddies/rapidmidiex/internal/suid"
)

type Client struct {
	mu sync.RWMutex

	ps map[suid.UUID]*Pool
}

var DefaultClient = &Client{
	ps: make(map[suid.UUID]*Pool),
}

func NewClient() *Client {
	c := &Client{
		ps: make(map[suid.UUID]*Pool),
	}

	return c
}

func (c *Client) Size() int { return len(c.ps) }

func (c *Client) Close() error {
	return rmx.ErrTodo
}

func (c *Client) NewPool() (suid.UUID, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	p := DefaultPool()

	c.ps[p.ID] = p

	return p.ID, nil
}

func (c *Client) Get(uid suid.UUID) (*Pool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for id, p := range c.ps {
		if id == uid {
			return p, nil
		}
	}

	return nil, ErrNoPool
}

func (c *Client) Has(uid suid.UUID) bool {
	_, err := c.Get(uid)
	return err == nil
}
