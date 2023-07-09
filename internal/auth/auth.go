package auth

import (
	"net/http"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/rapidmidiex/rmx/internal/sessions"
)

var (
	KeysetURL   string
	KeysetCache *jwk.Cache
)

func IsAuthenticated(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, err := sessions.Default(r)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		session, err := sess.Get(r)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if session.Profile == nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		keyset, err := KeysetCache.Get(r.Context(), KeysetURL)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if _, err := jwt.Parse([]byte(session.AccessToken), jwt.WithKeySet(keyset)); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		next(w, r)
	}
}
