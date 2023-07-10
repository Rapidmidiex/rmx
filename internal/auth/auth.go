package auth

import (
	"net/http"

	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
)

var ValidatorM *jwtmiddleware.JWTMiddleware

func IsAuthenticated(next http.Handler) http.Handler {
	return ValidatorM.CheckJWT(next)
}
