package middlewares

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/nats-io/nats.go"
	"github.com/rapidmidiex/rmx/internal/events"
	"github.com/zitadel/oidc/v2/pkg/oidc"
)

type ctxKey string

var (
	SessionCtx ctxKey = "rmx.session"
	TokenCtx   ctxKey = "rmx.token"
)

type ParsedClaims struct {
	JwtID      string    `json:"jti"`
	Subject    string    `json:"sub"`
	Issuer     string    `json:"iss"`
	Audience   []string  `json:"aud"`
	IssuedAt   time.Time `json:"iat"`
	NotBefore  time.Time `json:"nbf"`
	Expiration time.Time `json:"exp"`
	Email      string    `json:"email"`
}

func VerifySession(next http.HandlerFunc, nc *nats.Conn, pubk *ecdsa.PublicKey) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), oidc.PrefixBearer)

		res, err := nc.Request(fmt.Sprint(
			events.NatsSubj,
			events.NatsSessionSufx,
			events.NatsIntrospectSufx,
		), []byte(token), time.Second*5)
		if err != nil {
			log.Printf("rmx: verify session\n%v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if string(res.Data) != events.TokenAccepted {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		parsed, err := jwt.Parse([]byte(token), jwt.WithKey(jwa.ES256, pubk))
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		emailInterface, ok := parsed.Get("email")
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		email, ok := emailInterface.(string)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		tokClaims := ParsedClaims{
			JwtID:      parsed.JwtID(),
			Subject:    parsed.Subject(),
			Issuer:     parsed.Issuer(),
			Audience:   parsed.Audience(),
			IssuedAt:   parsed.IssuedAt(),
			NotBefore:  parsed.NotBefore(),
			Expiration: parsed.Expiration(),
			Email:      email,
		}

		bs, err := json.Marshal(tokClaims)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		rCtx := r.WithContext(context.WithValue(r.Context(), SessionCtx, bs))
		next(w, rCtx)
	}
}
