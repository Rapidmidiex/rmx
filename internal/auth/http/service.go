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

func New(ctx context.Context) *Service {
	// TODO: initialise OAuth services here

	s := Service{
		mux: service.New(),
	}

	s.routes()
	return &s
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Service) routes() {
	for _, p := range s.Providers {
		s.mux.Mount("/", p.Handle())
	}
}
