package service

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	service "github.com/rapidmidiex/rmx/internal/http"
	"github.com/rapidmidiex/rmx/internal/jam"
	jamDB "github.com/rapidmidiex/rmx/internal/jam/postgres"
)

type Option func(*Service)

func WithBroker(ctx context.Context, cap uint) Option {
	return func(s *Service) {
		s.wsb = jam.NewBroker()
	}
}

type Service struct {
	mux service.Service

	wsb  jam.Broker
	repo jamDB.Repo
}

// NOTE broker should be a dependency
func New(ctx context.Context, r jamDB.Repo) *Service {
	s := Service{
		mux:  service.New(),
		repo: r,
		wsb:  jam.NewBroker(),
	}
	s.routes()
	return &s
}

// NewService returns a service that handles Jam sessions
func NewService() *Service {
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
	// TODO -- delete jam
	// TODO -- edit jam
	s.mux.Get("/", s.handleIndex())
	// s.mux.Post("/jams", s.handleCreateJam())
	// s.mux.Get("/jams/{uuid}", s.handleGetJam())
	// s.mux.Get("/jams/{uuid}/ws", s.handleP2PConn())
}

func parseUUID(r *http.Request) (uuid.UUID, error) {
	p := chi.URLParam(r, "uuid")
	return uuid.Parse(p)
}
