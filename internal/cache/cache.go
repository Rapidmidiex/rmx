package cache

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
	"time"

	"github.com/nats-io/nats.go"
)

type Cache struct {
	nkv    nats.KeyValue
	encKey []byte
}

func New(name string, conn *nats.Conn, ttl time.Duration, encKey []byte) (*Cache, error) {
	if encKey != nil && len(encKey) != 32 {
		return nil, errors.New("rmx: incompatible cache encryption key")
	}

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

	return &Cache{kv, encKey}, nil
}

func (c *Cache) Set(key string, value []byte) error {
	val := value
	if c.encKey != nil {
		v, err := c.encrypt(value)
		if err != nil {
			return err
		}

		val = v
	}

	_, err := c.nkv.Put(key, val)
	return err
}

func (c *Cache) Get(key string) ([]byte, error) {
	entry, err := c.nkv.Get(key)
	if err != nil {
		return nil, err
	}

	val := entry.Value()
	if c.encKey != nil {
		v, err := c.decrypt(val)
		if err != nil {
			return nil, err
		}

		val = v
	}

	return val, err
}

func (c *Cache) Delete(key string) error {
	return c.nkv.Delete(key)
}

// encryption code borrowed from here: https://github.com/gtank/cryptopasta/blob/master/encrypt.go
func (c *Cache) encrypt(value []byte) ([]byte, error) {
	block, err := aes.NewCipher(c.encKey[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, value, nil), nil
}

func (c *Cache) decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(c.encKey[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < gcm.NonceSize() {
		return nil, errors.New("rmx: malformed ciphertext")
	}

	return gcm.Open(
		nil,
		ciphertext[:gcm.NonceSize()],
		ciphertext[gcm.NonceSize():],
		nil,
	)
}
