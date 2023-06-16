package cache

import (
	"time"

	"github.com/nats-io/nats.go"
)

type Cache struct {
	nkv nats.KeyValue
}

func New(name string, conn *nats.Conn, ttl time.Duration) (*Cache, error) {
	js, err := conn.JetStream()
	if err != nil {
		return nil, err
	}

	kv, err := js.CreateKeyValue(&nats.KeyValueConfig{
		Bucket: name,
		TTL:    ttl,
	})
	if err != nil {
		return nil, err
	}

	return &Cache{kv}, nil
}

func (c *Cache) Set(key string, value []byte) error {
	_, err := c.nkv.Put(key, value)
	return err
}

func (c *Cache) Get(key string) ([]byte, error) {
	entry, err := c.nkv.Get(key)
	if err != nil {
		return nil, err
	}
	return entry.Value(), err
}

func (c *Cache) Delete(key string) error {
	return c.nkv.Delete(key)
}
