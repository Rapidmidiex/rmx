package auth

import (
	"github.com/go-chi/chi/v5"
)

var (
	authURI     = "/google"
	callbackURI = "/google/callback"
	scopes      = []string{"email", "profile", "openid"}
)

func NewGoogle(router chi.Router, clientID, clientSecret string) (*Service, error) {
	handlers, err := NewWithProvider(
		"https://accounts.google.com",
		clientID,
		clientSecret,
		callbackURI,
		scopes,
	)
	if err != nil {
		return nil, err
	}

	return &Service{handlers, authURI, callbackURI, router}, nil
}
