package service

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/rog-golang-buddies/rmx/service/jam"
	"github.com/rog-golang-buddies/rmx/service/user"
)

type Service struct {
	m chi.Router
	l *log.Logger
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.m.ServeHTTP(w, r) }

func New(r chi.Router) *Service {
	s := &Service{r, log.Default()}
	s.m.Use(middleware.Logger)

	// NOTE unsure how much is gained using a goroutine
	// will have to investigate
	go jam.NewService(s.m)
	go user.NewService(s.m)
	return s
}

func Default() *Service {
	return New(chi.NewMux())
}
