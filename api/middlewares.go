package api

import (
	"context"
	"net/http"
	"strings"
)

type ctxKey string

var emailCtxKey = ctxKey("email")

func (s *AuthService) CheckAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		at := r.Header.Get(authorizationHeader)
		bearer := strings.Split(at, " ")
		if !(len(bearer) > 1) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		userInfo, err := parseAccessTokenWithValidate(bearer[1])
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		privateClaims := userInfo.PrivateClaims()
		email, ok := privateClaims["email"].(string)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), emailCtxKey, email)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
