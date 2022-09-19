package api

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	h "github.com/hyphengolang/prelude/http"

	ws "github.com/rog-golang-buddies/rapidmidiex/api/websocket"
	"github.com/rog-golang-buddies/rapidmidiex/internal/suid"
)

type Service struct {
	r chi.Router
	l *log.Logger

	c *ws.Client
}

func (s Service) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.r.ServeHTTP(w, r) }

func NewService(r chi.Router) *Service {
	s := &Service{r, log.Default(), ws.DefaultClient}
	s.routes()
	return s
}

func (s Service) respond(w http.ResponseWriter, r *http.Request, data any, status int) {
	h.Respond(w, r, data, status)
}

func (s Service) fileServer(prefix string, dirname string) http.Handler {
	return h.FileServer(prefix, dirname)
}

func (S Service) parseUUID(w http.ResponseWriter, r *http.Request, key string) (suid.UUID, error) {
	return suid.ParseString(chi.URLParam(r, key))
}
