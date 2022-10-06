package service

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/rog-golang-buddies/rmx/internal/dto"
	"github.com/rog-golang-buddies/rmx/service/jam"
	"github.com/rog-golang-buddies/rmx/service/user"

	"github.com/rog-golang-buddies/rmx/test/mock"
)

type Service struct {
	m chi.Router
	l *log.Logger
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.m.ServeHTTP(w, r) }

func New(m chi.Router, ur dto.UserRepo) *Service {
	s := &Service{m, log.Default()}

	// NOTE unsure how much is gained using a goroutine
	// will have to investigate
	go jam.NewService(s.m)
	go user.NewService(s.m, ur)

	s.m.Use(middleware.Logger)
	return s
}

func Default() *Service {
	return New(chi.NewMux(), mock.UserRepo())
}
