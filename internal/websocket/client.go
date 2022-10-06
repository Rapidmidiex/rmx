package websocket

import (
	"log"
	"sync"

	"github.com/rog-golang-buddies/rmx/internal/dto"
	"github.com/rog-golang-buddies/rmx/internal/suid"
	"golang.org/x/exp/maps"
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
	return dto.ErrNotImplemented
}

func (c *Client) NewPool(maxCount int) (suid.UUID, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	p := NewPool(maxCount)

	c.ps[p.ID] = p

	return p.ID, nil
}

func (c *Client) Get(uid suid.UUID) (*Pool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for id, p := range c.ps {
		log.Println("is match?", id, uid, id == uid)
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

func (c *Client) List() []*Pool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return maps.Values(c.ps)
}
