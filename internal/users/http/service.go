package http

import (
	"net/http"

	"github.com/google/uuid"
	service "github.com/rapidmidiex/rmx/internal/http"
	"github.com/rapidmidiex/rmx/internal/users"
	"github.com/rapidmidiex/rmx/internal/users/repo"
)

type Option func(*Service)

func WithRepo(r repo.Repo) Option {
	return func(s *Service) {
		s.repo = r
	}
}

type Service struct {
	mux  service.Service
	repo repo.Repo
}

func New(opts ...Option) *Service {
	s := Service{
		mux: service.New(),
	}

	for _, opt := range opts {
		opt(&s)
	}

	// TODO -- This will be changed to panic once
	// I have implemented the postgres repo
	if s.repo == nil {
		s.repo = repo.New()
	}

	s.routes()
	return &s
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Service) routes() {
	s.mux.Post("/", s.handleCreateUser())
}

func (s *Service) handleCreateUser() http.HandlerFunc {
	type request struct {
		Username string `json:"username"`
	}

	type response struct {
		User *users.User `json:"user"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := s.mux.Decode(w, r, &req); err != nil {
			s.mux.Respond(w, r, err, http.StatusBadRequest)
			return
		}

		dto := users.User{
			ID:       uuid.New(),
			Username: req.Username,
		}

		created, err := s.repo.CreateUser(r.Context(), dto)
		if err != nil {
			s.mux.Respond(w, r, err, http.StatusInternalServerError)
			return
		}

		res := response{User: &created}
		s.mux.Respond(w, r, res, http.StatusCreated)
	}
}
