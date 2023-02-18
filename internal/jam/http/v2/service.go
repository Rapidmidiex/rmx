package service

import (
	"context"
	"io"
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
	mux service.Service

	store store
	wsb   jam.Broker
}

// NOTE broker should be a dependency
func NewService(ctx context.Context, store store) *Service {
	s := Service{
		mux:   service.New(),
		store: store,
		wsb:   jam.NewBroker(),
	}
	s.routes()
	return &s
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Service) routes() {
	s.mux.Post("/api/v1/jam", s.handleCreateJam())
	s.mux.Get("/api/v1/jam", s.handleListJams())
	s.mux.Get("/api/v1/jam/{uuid}", s.handleGetJam())
	s.mux.Get("/ws/jam/{uuid}", s.handleP2PConn())
}

func (s *Service) handleCreateJam() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var j jam.Jam
		if err := s.mux.Decode(w, r, &j); err != nil && err != io.EOF {
			s.mux.Logf("decode: %v\n", err)
			s.mux.Respond(w, r, err, http.StatusBadRequest)
			return
		}

		j.SetDefaults()

		created, err := s.store.CreateJam(r.Context(), j)
		if err != nil {
			s.mux.Logf("createJam: %v\n", err)
			s.mux.Respond(w, r, err, http.StatusInternalServerError)
			return
		}

		// s.br.Store(created.ID.String(), &created)
		s.mux.Respond(w, r, created, http.StatusCreated)
	}
}

func (s *Service) handleGetJam() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// NOTE move to middleware
		jamID, err := parseUUID(r)
		if err != nil {
			s.mux.Logf("parseUUID: %v\n", err)
			s.mux.Respond(w, r, jamID, http.StatusBadRequest)
			return
		}

		jam, err := s.store.GetJamByID(r.Context(), jamID)
		if err != nil {
			s.mux.Logf("getJamByID: %v\n", err)
			s.mux.Respond(w, r, err, http.StatusNotFound)
			return
		}

		s.mux.Respond(w, r, jam, http.StatusOK)
	}
}

func (s *Service) handleListJams() http.HandlerFunc {
	type roomResp struct {
		jam.Jam
		PlayerCount int `json:"playerCount"`
	}
	type response struct {
		Rooms []roomResp `json:"rooms"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		jams, err := s.store.GetJams(r.Context())
		if err != nil {
			s.mux.Logf("getJams: %v", err)
			s.mux.Respond(w, r, err, http.StatusInternalServerError)
			return
		}

		var resp response
		resp.Rooms = make([]roomResp, 0)
		for _, j := range jams {
			resp.Rooms = append(resp.Rooms, roomResp{
				Jam:         j,
				PlayerCount: -1, // TODO fix this later
			})
		}

		s.mux.Respond(w, r, resp, http.StatusOK)
	}
}

func (s *Service) handleP2PConn() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// NOTE move to middleware
		uid, err := parseUUID(r)
		if err != nil {
			s.mux.Respond(w, r, uid, http.StatusBadRequest)
			return
		}

		jam, err := s.store.GetJamByID(r.Context(), uid)
		if err != nil {
			s.mux.Respond(w, r, err, http.StatusNotFound)
			return
		}

		// get from websocket client
		loaded, _ := s.wsb.LoadOrStore(jam.ID.String(), &jam)
		loaded.Client().ServeHTTP(w, r)
	}
}

func parseUUID(r *http.Request) (uuid.UUID, error) {
	p := chi.URLParam(r, "uuid")
	return uuid.Parse(p)
}

type Option func(*Service)

func WithBroker(ctx context.Context, cap uint) Option {
	return func(s *Service) {
		s.wsb = jam.NewBroker()
	}
}
