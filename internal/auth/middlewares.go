package auth

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"net/http"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

type sessionCtxKey struct{}

type Session struct {
	SID    string
	Issuer string
	Email  string
}

// set allowUnauthorized to false if you want to blocke access without authorization
func VerifySession(next http.HandlerFunc, pubk *ecdsa.PublicKey, allowUnauthorized bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parsed, err := jwt.Parse([]byte(r.Header.Get("Authorization")), jwt.WithKey(jwa.ES256, pubk))
		if err != nil {
			// pass the request directly without modification
			if allowUnauthorized {
				next(w, r)
				return
			}

			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		rCtx := r.WithContext(context.WithValue(r.Context(), sessionCtxKey{}, Session{
			SID:    parsed.JwtID(),
			Issuer: parsed.Issuer(),
			Email:  parsed.Subject(),
		}))

		next(w, rCtx)
	}
}

// GetSessionFromContext returns an error if the session is `nil` or invalid
func GetSessionFromContext(ctx context.Context) (*Session, error) {
	sess, ok := ctx.Value(sessionCtxKey{}).(Session)
	if !ok {
		return nil, errors.New("invalid session")
	}

	return &sess, nil
}
