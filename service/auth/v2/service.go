package auth

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	h "github.com/hyphengolang/prelude/http"

	"github.com/rog-golang-buddies/rmx/internal/dto"
	"github.com/rog-golang-buddies/rmx/internal/suid"
	// big no-no
)

var (
	ErrNoCookie        = errors.New("user: cookie not found")
	ErrSessionNotFound = errors.New("user: session not found")
	ErrSessionExists   = errors.New("user: session already exists")
)

/*
Register a new user

	[ ] POST /auth/register

Create a cookie

	[ ] POST /auth/login

Delete a cookie

	[ ] DELETE /auth/logout

Refresh token

	[ ] GET /auth/refresh
*/
func (s *Service) routes() {
	s.m.Route("/api/v2/auth", func(r chi.Router) {
		r.Post("/signup", s.handleSignUp())
	})
}

func (s *Service) handleSignUp() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.respond(w, r, nil, http.StatusNotImplemented)
	}
}

type Service struct {
	ctx context.Context

	m chi.Router

	log  func(...any)
	logf func(string, ...any)

	decode    func(http.ResponseWriter, *http.Request, any) error
	respond   func(http.ResponseWriter, *http.Request, any, int)
	created   func(http.ResponseWriter, *http.Request, string)
	setCookie func(http.ResponseWriter, *http.Cookie)
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.m.ServeHTTP(w, r) }

func NewService(ctx context.Context, m chi.Router, r dto.UserRepo) *Service {
	s := &Service{
		ctx,
		m,

		log.Println,
		log.Printf,

		h.Decode,
		h.Respond,
		h.Created,
		http.SetCookie,
	}

	s.routes()
	return s
}

func (s *Service) parseUUID(w http.ResponseWriter, r *http.Request) (suid.UUID, error) {
	return suid.ParseString(chi.URLParam(r, "uuid"))
}
