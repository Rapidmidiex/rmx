package auth

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	h "github.com/hyphengolang/prelude/http"

	"github.com/rog-golang-buddies/rmx/internal"
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
		// tokens
	})

	s.m.Route("/api/v2/account", func(r chi.Router) {
		r.Post("/signup", s.handleSignUp())
	})
}

func (s *Service) handleSignUp() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var u internal.User
		if err := s.newUser(w, r, &u); err != nil {
			s.respond(w, r, err, http.StatusBadRequest)
			return
		}

		if err := s.r.Insert(r.Context(), &u); err != nil {
			s.respond(w, r, err, http.StatusInternalServerError)
			return
		}

		s.created(w, r, u.ID.ShortUUID().String())
	}
}

type Service struct {
	ctx context.Context

	m chi.Router
	r internal.WUserRepo

	log  func(...any)
	logf func(string, ...any)

	decode    func(http.ResponseWriter, *http.Request, any) error
	respond   func(http.ResponseWriter, *http.Request, any, int)
	created   func(http.ResponseWriter, *http.Request, string)
	setCookie func(http.ResponseWriter, *http.Cookie)
}

func (s *Service) newUser(w http.ResponseWriter, r *http.Request, u *internal.User) (err error) {
	var dto User
	if err = s.decode(w, r, &dto); err != nil {
		return
	}

	var h internal.PasswordHash
	h, err = dto.Password.Hash()
	if err != nil {
		return
	}

	*u = internal.User{
		ID:       suid.NewUUID(),
		Username: dto.Username,
		Email:    dto.Email,
		Password: h,
	}

	return nil
}

func (s *Service) parseUUID(w http.ResponseWriter, r *http.Request) (suid.UUID, error) {
	return suid.ParseString(chi.URLParam(r, "uuid"))
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.m.ServeHTTP(w, r) }

func NewService(ctx context.Context, m chi.Router, r internal.WUserRepo) *Service {
	s := &Service{
		ctx,

		m,
		r,

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

type User struct {
	Email    internal.Email    `json:"email"`
	Username string            `json:"username"`
	Password internal.Password `json:"password"`
}
