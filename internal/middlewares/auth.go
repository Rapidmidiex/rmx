package middlewares

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rapidmidiex/rmx/internal/events"
)

type ctxKey string

var sessionCtx ctxKey = "rmx.session"

func ParseSession(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		split := strings.Split(token, "Bearer")
		if len(split) != 2 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		token = strings.TrimSpace(split[1])
		rCtx := r.WithContext(context.WithValue(r.Context(), sessionCtx, token))
		next(w, rCtx)
	}
}

func VerifySession(next http.HandlerFunc, nc *nats.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := r.Context().Value(sessionCtx).(string)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

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

		if string(res.Data) == events.TokenAccepted {
			next(w, r)
		}

		w.WriteHeader(http.StatusUnauthorized)
	}
}
