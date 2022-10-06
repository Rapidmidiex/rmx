package user

import (
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	h "github.com/hyphengolang/prelude/http"

	"github.com/rog-golang-buddies/rmx/internal/dto"
	"github.com/rog-golang-buddies/rmx/internal/suid"
	"github.com/rog-golang-buddies/rmx/test/mock" // big no-no
)

var (
	ErrNoCookie        = errors.New("user: cookie not found")
	ErrSessionNotFound = errors.New("user: session not found")
	ErrSessionExists   = errors.New("user: session already exists")
)

type Service struct {
	m  chi.Router
	ur dto.UserRepo

	l *log.Logger
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.m.ServeHTTP(w, r) }

func NewService(m chi.Router, r dto.UserRepo) *Service {
	s := &Service{m: m, ur: r, l: log.Default()}
	s.routes()
	return s
}

func DefaultService() *Service {
	s := &Service{m: chi.NewMux(), ur: mock.UserRepo(), l: log.Default()}
	s.routes()
	return s
}

func (s *Service) respond(w http.ResponseWriter, r *http.Request, data any, status int) {
	h.Respond(w, r, data, status)
}

func (s *Service) respondCookie(w http.ResponseWriter, r *http.Request, data any, c *http.Cookie) {
	http.SetCookie(w, c)
	s.respond(w, r, data, http.StatusOK)
}

func (s *Service) created(w http.ResponseWriter, r *http.Request, id string) {
	h.Created(w, r, id)
}

func (s *Service) decode(w http.ResponseWriter, r *http.Request, data interface{}) error {
	return h.Decode(w, r, data)
}

func (s *Service) parseUUID(w http.ResponseWriter, r *http.Request) (suid.UUID, error) {
	return suid.ParseString(chi.URLParam(r, "uuid"))
}

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
