package websocket

import (
	"sync"

	"github.com/google/uuid"

	rmx "github.com/rog-golang-buddies/rapidmidiex/internal"
)

type Client struct {
	mu sync.Mutex

	ps map[uuid.UUID]*Pool
}

var DefaultClient = &Client{
	ps: make(map[uuid.UUID]*Pool),
}

func NewClient() *Client {
	c := &Client{
		ps: make(map[uuid.UUID]*Pool),
	}

	return c
}

func (c *Client) Size() int { return len(c.ps) }

func (c *Client) Close() error {
	return rmx.ErrTodo
}

func (c *Client) NewPool() (uuid.UUID, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	p := DefaultPool()

	c.ps[p.ID] = p

	return p.ID, nil
}

func (c *Client) Get(uid uuid.UUID) (*Pool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for id, p := range c.ps {
		if id == uid {
			return p, nil
		}
	}

	return nil, ErrNoPool
}

func (c *Client) Has(uid uuid.UUID) bool {
	_, err := c.Get(uid)
	return err == nil
}
