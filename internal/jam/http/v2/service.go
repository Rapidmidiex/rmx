package service

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	service "github.com/rapidmidiex/rmx/internal/http"
	"github.com/rapidmidiex/rmx/internal/jam"
)

// TODO this needs to be moved into the repository pacakge
type store interface {
	CreateJam(context.Context, jam.Jam) (jam.Jam, error)
	GetJams(context.Context) ([]jam.Jam, error)
	GetJamByID(ctx context.Context, id uuid.UUID) (jam.Jam, error)
}

type Service struct {
	m service.Service

	s store
}

func New() *Service {
	s := &Service{}
	s.routes()
	return s
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.m.ServeHTTP(w, r)
}

func (s *Service) routes() {
	s.m.Get("/api/v1/jam", http.NotFound)
	s.m.Post("/api/v1/jam", http.NotFound)
	s.m.Get("/api/v1/jam/{uuid}", http.NotFound)

	s.m.Get("/ws/jam/{uuid}", http.NotFound)
}

func (s *Service) handleCreateJam() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO
	}
}
