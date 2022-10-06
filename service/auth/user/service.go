package user

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

// source: https://stackoverflow.com/questions/7140074/restfully-design-login-or-register-resources
//
//	[+] GET /user/me - get my user details
//	[-] GET /user/{uuid} - get user info
//	[-] GET /user - get all users
//	[+] POST /user - register new user
//	[ ] POST /user/{uuid} - update information about user
//	[ ] DELETE /user - delete user from database
//	[+] GET /session - refresh session token
//	[+] POST /session - create session (due to logging in)
//	[+] DELETE /session - delete session (due to logging out)
func (s *Service) routes() {}

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

func NewService(ctx context.Context, m chi.Router, r internal.UserRepo) *Service {
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
