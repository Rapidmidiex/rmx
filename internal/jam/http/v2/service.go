package service

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
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

	r  store
	br jam.Broker
}

// NOTE broker should be a dependency
func New(ctx context.Context, store store) *Service {
	s := &Service{
		m:  service.New(),
		r:  store,
		br: jam.NewBroker(),
	}
	s.routes()
	return s
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.m.ServeHTTP(w, r)
}

func (s *Service) routes() {
	s.m.Post("/api/v1/jam", s.handleCreateJam())
	// s.m.Get("/api/v1/jam", http.NotFound)
	s.m.Get("/api/v1/jam/{uuid}", s.handleGetJam())

	s.m.Get("/ws/jam/{uuid}", s.handleP2PConn())
}

func (s *Service) handleCreateJam() http.HandlerFunc {
	type Q struct {
		Name     string `json:"name"`
		Capacity uint   `json:"capacity"`
		BPM      uint   `json:"bpm"`
	}

	newJam := func(w http.ResponseWriter, r *http.Request) (jam.Jam, error) {
		var q Q
		if err := s.m.Decode(w, r, &q); err != nil {
			return jam.Jam{}, err
		}

		j := jam.Jam{
			Name:     q.Name,
			Capacity: q.Capacity,
			BPM:      q.BPM,
		}
		return j, nil
	}

	return func(w http.ResponseWriter, r *http.Request) {
		j, err := newJam(w, r)
		if err != nil {
			s.m.Respond(w, r, err, http.StatusBadRequest)
			return
		}

		created, err := s.r.CreateJam(r.Context(), j)
		if err != nil {
			s.m.Respond(w, r, err, http.StatusInternalServerError)
			return
		}

		s.br.Store(created.ID.String(), &created)

		s.m.Respond(w, r, created, http.StatusCreated)
	}
}

func (s *Service) handleGetJam() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// NOTE move to middleware
		uid, err := parseUUID(r)
		if err != nil {
			s.m.Respond(w, r, uid, http.StatusBadRequest)
			return
		}

		jam, err := s.r.GetJamByID(r.Context(), uid)
		if err != nil {
			s.m.Respond(w, r, err, http.StatusNotFound)
			return
		}

		s.m.Respond(w, r, jam, http.StatusOK)
	}
}

func (s *Service) handleP2PConn() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// NOTE move to middleware
		uid, err := parseUUID(r)
		if err != nil {
			s.m.Respond(w, r, uid, http.StatusBadRequest)
			return
		}

		jam, err := s.r.GetJamByID(r.Context(), uid)
		if err != nil {
			s.m.Respond(w, r, err, http.StatusNotFound)
			return
		}

		// get from websocket client
		loaded, _ := s.br.LoadOrStore(jam.ID.String(), &jam)
		loaded.Client().ServeHTTP(w, r)
	}
}

func parseUUID(r *http.Request) (uuid.UUID, error) {
	p := chi.URLParam(r, "uuid")
	return uuid.Parse(p)
}
