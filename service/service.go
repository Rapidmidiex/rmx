package service

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Service struct {
	m chi.Router
	l *log.Logger
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.m.ServeHTTP(w, r) }

func New(r chi.Router) *Service {
	s := &Service{r, log.Default()}
	return s
}

func Default() *Service {
	s := &Service{chi.NewMux(), log.Default()}
	//
	return s
}
