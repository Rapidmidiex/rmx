package service

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	h "github.com/hyphengolang/prelude/http"
)

type Service interface {
	chi.Router

	Log(...any)
	Logf(string, ...any)

	Decode(http.ResponseWriter, *http.Request, any) error
	Respond(http.ResponseWriter, *http.Request, any, int)
	RespondText(w http.ResponseWriter, r *http.Request, status int)
	Created(http.ResponseWriter, *http.Request, string)
	SetCookie(http.ResponseWriter, *http.Cookie)
}

type service struct {
	chi.Router
}

// Created implements Service
func (*service) Created(w http.ResponseWriter, r *http.Request, id string) {
	h.Created(w, r, id)
}

// Decode implements Service
func (*service) Decode(w http.ResponseWriter, r *http.Request, v any) error {
	return h.Decode(w, r, v)
}

// Log implements Service
func (*service) Log(v ...any) { log.Println(v...) }

// Logf implements Service
func (*service) Logf(format string, v ...any) { log.Printf(format, v...) }

// Respond implements Service
func (*service) Respond(w http.ResponseWriter, r *http.Request, v any, status int) {
	h.Respond(w, r, v, status)
}

func (s *service) RespondText(w http.ResponseWriter, r *http.Request, status int) {
	s.Respond(w, r, http.StatusText(status), status)
}

// SetCookie implements Service
func (*service) SetCookie(w http.ResponseWriter, c *http.Cookie) { http.SetCookie(w, c) }

func New(opt ...Option) Service {
	var s service
	for _, o := range opt {
		o(&s)
	}

	if s.Router == nil {
		s.Router = chi.NewRouter()
	}

	return &s
}

type Option func(*service)

func WithRouter(mux chi.Router) Option {
	return func(s *service) {
		s.Router = mux
	}
}
