package sessions

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"time"
)

// TODO: use global name for sessions and allow adding fields to session object
type Store struct {
	ttl    time.Duration
	encKey []byte
}

func New(ttl time.Duration, encKey []byte) (*Store, error) {
	if encKey != nil && len(encKey) != 32 {
		return nil, errors.New("rmx: incompatible session encryption key")
	}

	return &Store{ttl, encKey}, nil
}

func (s *Store) Set(w http.ResponseWriter, name string, value string) error {
	val := value
	if s.encKey != nil {
		v, err := s.encrypt([]byte(value))
		if err != nil {
			return err
		}

		val = string(v)
	}
	val = base64.StdEncoding.EncodeToString([]byte(val))

	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    val,
		Path:     "/",
		Expires:  time.Now().Add(s.ttl),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	return nil
}

func (s *Store) Get(r *http.Request, name string) (string, error) {
	cookie, err := r.Cookie(name)
	if err != nil {
		return "", err
	}

	decoded, err := base64.StdEncoding.DecodeString(cookie.Value)
	if err != nil {
		return "", err
	}

	if s.encKey != nil {
		v, err := s.decrypt(decoded)
		if err != nil {
			return "", err
		}

		decoded = v
	}

	return string(decoded), nil
}

// encryption code borrowed from here: https://github.com/gtank/cryptopasta/blob/master/encrypt.go
func (s *Store) encrypt(value []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.encKey[:])
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

func (s *Store) decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.encKey[:])
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
