package auth

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Provider struct {
	AuthHandler, CallbackHandler http.HandlerFunc
	AuthURI, CallbackURI         string
}

func (p *Provider) Handle() chi.Router {
	return chi.NewMux().Route("/", func(r chi.Router) {
		r.Handle(p.AuthURI, p.AuthHandler)
		r.Handle(p.CallbackURI, p.CallbackHandler)
	})
}

type AuthError struct {
	StatusCode int   `json:"status"`
	Err        error `json:"err"`
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("status %d: err %v", e.StatusCode, e.Err)
}
