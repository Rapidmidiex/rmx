package service

import (
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	service "github.com/rapidmidiex/rmx/internal/http"
	"github.com/rapidmidiex/rmx/internal/jams"
	jamsDB "github.com/rapidmidiex/rmx/internal/jams/postgres"
	"github.com/rapidmidiex/rmx/pkg/fp"
)

type Option func(*Service)

func WithRepo(r jamsDB.Repo) Option {
	return func(s *Service) {
		s.repo = r
	}
}

type Service struct {
	mux  service.Service
	repo jamsDB.Repo
	wsb  jams.Broker
}

// NOTE broker should be a dependency
// func New(ctx context.Context, r jamDB.Repo) *Service {
func New(opts ...Option) *Service {
	s := Service{
		mux: service.New(),
		wsb: jams.NewBroker(),
	}

	for _, opt := range opts {
		opt(&s)
	}

	if s.repo == nil {
		panic("repo is nil")
	}

	s.routes()
	return &s
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Service) routes() {
	s.mux.Post("/", s.handleCreateJam())
	s.mux.Get("/", s.handleListJams())
	s.mux.Get("/{uuid}", s.handleGetJam())

	s.mux.Get("/{uuid}/ws", s.handleP2PConn())
}

func (s *Service) handleCreateJam() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var dto jams.Jam
		if err := s.mux.Decode(w, r, &dto); err != nil && err != io.EOF {
			s.mux.Logf("decode: %v\n", err)
			s.mux.Respond(w, r, err, http.StatusBadRequest)
			return
		}

		dto.SetDefaults()

		created, err := s.repo.CreateJam(r.Context(), dto)
		if err != nil {
			s.mux.Logf("createJam: %v\n", err)
			s.mux.Respond(w, r, err, http.StatusInternalServerError)
			return
		}

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

		jam, err := s.repo.GetJamByID(r.Context(), jamID)
		if err != nil {
			s.mux.Logf("getJamByID: %v\n", err)
			s.mux.Respond(w, r, err, http.StatusNotFound)
			return
		}

		s.mux.Respond(w, r, jam, http.StatusOK)
	}
}

func (s *Service) handleListJams() http.HandlerFunc {
	type room struct {
		jams.Jam
		PlayerCount int `json:"playerCount"`
	}

	type response struct {
		Rooms []room `json:"rooms"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		found, err := s.repo.GetJams(r.Context())
		if err != nil {
			s.mux.Logf("getJams: %v", err)
			s.mux.Respond(w, r, err, http.StatusInternalServerError)
			return
		}

		resp := response{
			Rooms: fp.FMap(found, func(j jams.Jam) room {
				loaded, _ := s.wsb.LoadOrStore(j.ID, &j)
				return room{*loaded, loaded.Client().Len()}
			}),
		}

		s.mux.Respond(w, r, resp, http.StatusOK)
	}
}

func (s *Service) handleP2PConn() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// NOTE move to middleware
		jamID, err := parseUUID(r)
		if err != nil {
			s.mux.Respond(w, r, jamID, http.StatusBadRequest)
			return
		}

		found, err := s.repo.GetJamByID(r.Context(), jamID)
		if err != nil {
			s.mux.Respond(w, r, err, http.StatusNotFound)
			return
		}

		// get from websocket client
		jam, _ := s.wsb.LoadOrStore(found.ID, &found)
		jam.Client().ServeHTTP(w, r)
	}
}

func parseUUID(r *http.Request) (uuid.UUID, error) {
	p := chi.URLParam(r, "uuid")
	return uuid.Parse(p)
}
