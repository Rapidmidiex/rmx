package auth

import (
	"fmt"
	"net/http"

	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/rapidmidiex/rmx/internal/sessions"
)

var Validator *validator.Validator

func IsAuthenticated(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := sessions.GetSession(r)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if session.Profile == nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		res, err := Validator.ValidateToken(r.Context(), session.AccessToken)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		fmt.Println(res)

		next(w, r)
	}
}
