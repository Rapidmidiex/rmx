package service

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/rog-golang-buddies/rmx/store"
)

func (s *Service) routes() {
	s.m.Use(middleware.Logger)
}

type Service struct {
	m chi.Router

	log  func(s ...any)
	logf func(string, ...any)
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.m.ServeHTTP(w, r) }

func New(st store.Store) *Service {
	s := &Service{chi.NewMux(), log.Print, log.Printf}
	s.routes()
	return s
}
