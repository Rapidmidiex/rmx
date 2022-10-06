package middlewares

import (
	"context"
	"net/http"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

type contextKey string

var emailKey = contextKey("rmx-email")

func Authenticate(publicKey jwk.Key) func(f http.Handler) http.Handler {
	return func(f http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			// token, err := jwt.ParseRequest(r, jwt.WithHeaderKey("Authorization"), jwt.WithHeaderKey(cookieName), jwt.WithKey(jwa.RS256, publicKey), jwt.WithValidate(true))
			token, err := jwt.ParseRequest(r, jwt.WithKey(jwa.RS256, publicKey))
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			email, ok := token.PrivateClaims()["email"].(string)
			if !ok {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			// NOTE convert email from `string` type to `internal.Email` ?
			r = r.WithContext(context.WithValue(r.Context(), emailKey, email))
			f.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}
