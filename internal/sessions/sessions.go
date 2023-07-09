package sessions

import (
	"context"
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
	// NOTE state may be used only once
	State       string         `json:"state"`
	AccessToken string         `json:"accessToken"`
	Profile     map[string]any `json:"profile"`
}

type SessionsKey struct{}

func New(name string, ttl time.Duration, encKey []byte) (func(next http.Handler) http.Handler, error) {
	if encKey != nil && len(encKey) != 32 {
		return nil, errors.New("rmx: incompatible session encryption key")
	}

	s := &Store{name, ttl, encKey}

	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			rCtx := context.WithValue(r.Context(), SessionsKey{}, s)
			next.ServeHTTP(w, r.WithContext(rCtx))
		}

		return http.HandlerFunc(fn)
	}, nil
}

// NOTE rename FromContext(ctx *context.Contexttext)
// FromRequest(r *http.Request)

func SetSession(w http.ResponseWriter, r *http.Request, session *Session) error {
	s, err := storeFromRequest(r)
	if err != nil {
		return err
	}

	// NOTE (gob).Marshal({state, accessToken, profile})
	val, err := json.Marshal(session)
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

func GetSession(r *http.Request) (*Session, error) {
	s, err := storeFromRequest(r)
	if err != nil {
		return nil, err
	}

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

func RemoveFromRequest(w http.ResponseWriter, r *http.Request) error {
	s, err := storeFromRequest(r)
	if err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     s.name,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
	})

	return nil
}

func storeFromRequest(r *http.Request) (*Store, error) {
	sess, ok := r.Context().Value(SessionsKey{}).(*Store)
	if !ok {
		return nil, errors.New("rmx: no session store found in context")
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
