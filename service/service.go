package service

import (
	"context"
	"log"
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/rog-golang-buddies/rmx/service/auth"
	jam "github.com/rog-golang-buddies/rmx/service/jam/v2"
	"github.com/rog-golang-buddies/rmx/store"
)

func (s *Service) routes() {
	s.m.Use(middleware.Logger)
}

type Service struct {
	m chi.Router

	log    func(s ...any)
	logf   func(string, ...any)
	fatal  func(s ...any)
	fatalf func(string, ...any)
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.m.ServeHTTP(w, r) }

func New(ctx context.Context, st *store.Store) http.Handler {
	s := &Service{chi.NewMux(), log.Print, log.Printf, log.Fatal, log.Fatalf}

	s.routes()

	// TODO - use mux.Mount instead. But this works
	auth.NewService(ctx, s.m, st.UserRepo(), st.TokenClient())
	jam.NewService(ctx, s.m)

	return s
}
