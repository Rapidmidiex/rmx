package sessions

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"
)

// TODO: use global name for sessions and allow adding fields to session object
type Store struct {
	name   string
	ttl    time.Duration
	encKey []byte
}

type Session struct {
	State       string         `json:"state"`
	AccessToken string         `json:"accessToken"`
	Profile     map[string]any `json:"profile"`
}

func New(name string, ttl time.Duration, encKey []byte) (*Store, error) {
	if encKey != nil && len(encKey) != 32 {
		return nil, errors.New("rmx: incompatible session encryption key")
	}

	return &Store{name, ttl, encKey}, nil
}

func (s *Store) Set(w http.ResponseWriter, sess *Session) error {
	val, err := json.Marshal(sess)
	if err != nil {
		return err
	}

	if s.encKey != nil {
		v, err := s.encrypt(val)
		if err != nil {
			return err
		}

		val = v
	}

	http.SetCookie(w, &http.Cookie{
		Name:     s.name,
		Value:    base64.StdEncoding.EncodeToString(val),
		Path:     "/",
		Expires:  time.Now().Add(s.ttl),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	return nil
}

func (s *Store) Get(r *http.Request) (*Session, error) {
	cookie, err := r.Cookie(s.name)
	if err != nil {
		return nil, err
	}

	decoded, err := base64.StdEncoding.DecodeString(cookie.Value)
	if err != nil {
		return nil, err
	}

	if s.encKey != nil {
		v, err := s.decrypt(decoded)
		if err != nil {
			return nil, err
		}

		decoded = v
	}

	var sess *Session
	if err := json.Unmarshal(decoded, &sess); err != nil {
		return nil, err
	}

	return sess, nil
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
