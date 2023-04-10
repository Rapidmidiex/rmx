package http

import (
	"context"
	"net/http"

	"github.com/rapidmidiex/rmx/internal/auth"
	service "github.com/rapidmidiex/rmx/internal/http"
)

type Service struct {
	mux service.Service

	Providers []*auth.Provider
}

func New(ctx context.Context, providers ...*auth.Provider) *Service {
	s := Service{
		mux: service.New(),

		Providers: providers,
	}

	s.routes()
	return &s
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Service) routes() {
	for _, p := range s.Providers {
		s.mux.Handle(p.AuthURI, p.AuthHandler)
		s.mux.Handle(p.CallbackURI, p.CallbackHandler)
	}
}
