package service

import (
	"context"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	h "github.com/hyphengolang/prelude/http"
)

type Service interface {
	chi.Router

	Context() context.Context

	Log(...any)
	Logf(string, ...any)

	Decode(http.ResponseWriter, *http.Request, any) error
	Respond(http.ResponseWriter, *http.Request, any, int)
	RespondText(w http.ResponseWriter, r *http.Request, status int)
	Created(http.ResponseWriter, *http.Request, string)
	SetCookie(http.ResponseWriter, *http.Cookie)
}

type service struct {
	ctx context.Context
	chi.Router
}

// Context implements Service
func (s *service) Context() context.Context {
	if s.ctx == nil {
		return context.Background()
	}
	return s.ctx
}

// Created implements Service
func (*service) Created(w http.ResponseWriter, r *http.Request, id string) { h.Created(w, r, id) }

// Decode implements Service
func (*service) Decode(w http.ResponseWriter, r *http.Request, v any) error { return h.Decode(w, r, v) }

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

func New(ctx context.Context, mux chi.Router) Service {
	return &service{ctx, mux}
}
