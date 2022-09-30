package user

import (
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	h "github.com/hyphengolang/prelude/http"

	"github.com/rog-golang-buddies/rmx/internal"
	"github.com/rog-golang-buddies/rmx/internal/suid"
	"github.com/rog-golang-buddies/rmx/test/mock"
)

type contextKey string

func chain(hf http.HandlerFunc, mw ...h.MiddleWare) http.HandlerFunc { return h.Chain(hf, mw...) }

var (
	ErrNoCookie        = errors.New("api: cookie not found")
	ErrSessionNotFound = errors.New("api: session not found")
	ErrSessionExists   = errors.New("api: session already exists")
)

type User struct {
	ID   suid.SUID `json:"id"`
	Name string    `json:"name,omitempty"`
	/* More fields can belong here */
}

type Service struct {
	m chi.Router
	r internal.RUserRepo

	l *log.Logger
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.m.ServeHTTP(w, r) }

func NewService(m chi.Router, r internal.RUserRepo) *Service {
	s := &Service{m, r, log.Default()}
	s.routes()
	return s
}

func DefaultService() *Service {

	s := &Service{chi.NewMux(), mock.UserRepo(), log.Default()}
	s.routes()
	return s
}

func (s *Service) respond(w http.ResponseWriter, r *http.Request, data any, status int) {
	h.Respond(w, r, data, status)
}

func (s *Service) decode(w http.ResponseWriter, r *http.Request, data interface{}) error {
	return h.Decode(w, r, data)
}

func (s *Service) parseUUID(w http.ResponseWriter, r *http.Request) (suid.UUID, error) {
	return suid.ParseString(chi.URLParam(r, "id"))
}

func (s *Service) routes() {
	s.m.Route("/api/v1/user", func(r chi.Router) {
		r.Get("/tba", s.handleUserLogin())
		r.Post("/tba", s.handleUserSignUp())

		// health
		r.Get("/ping", s.handlePing)
	})
}

func (s *Service) handlePing(w http.ResponseWriter, r *http.Request) {
	s.respond(w, r, nil, http.StatusNoContent)
}
