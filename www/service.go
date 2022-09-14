package www

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	h "github.com/hyphengolang/prelude/http"

	"github.com/rog-golang-buddies/rapidmidiex/www/ws"
)

type Service struct {
	r chi.Router
	l *log.Logger

	p *ws.Pool
}

func (s Service) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.r.ServeHTTP(w, r) }

func NewService(r chi.Router) *Service {
	s := &Service{r, log.Default(), ws.DefaultPool()}
	s.routes()
	return s
}

func (s Service) respond(w http.ResponseWriter, r *http.Request, data interface{}, status int) {
	h.Respond(w, r, data, status)
}

func (s Service) fileServer(prefix string, dirname string) http.Handler {
	return h.FileServer(prefix, dirname)
}
