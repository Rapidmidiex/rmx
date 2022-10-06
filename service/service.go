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

	l *log.Logger
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.m.ServeHTTP(w, r) }

func New(st store.Store) *Service {
	s := &Service{chi.NewMux(), log.Default()}
	s.routes()

	// jam.NewService(s.m)
	// user.NewService(s.m, ur)
	return s
}
