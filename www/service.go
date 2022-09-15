package www

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	suid "github.com/lithammer/shortuuid/v4"

	h "github.com/hyphengolang/prelude/http"

	rmx "github.com/rog-golang-buddies/rapidmidiex/internal"
	ws "github.com/rog-golang-buddies/rapidmidiex/www/ws"
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

func (s Service) respond(w http.ResponseWriter, r *http.Request, data interface{}, status int) {
	h.Respond(w, r, data, status)
}

func (s Service) fileServer(prefix string, dirname string) http.Handler {
	return h.FileServer(prefix, dirname)
}

func (s Service) safeUUID(uid uuid.UUID) rmx.ID {
	return rmx.ID(suid.DefaultEncoder.Encode(uid))
}

func (S Service) parseUUID(w http.ResponseWriter, r *http.Request, key string) (uuid.UUID, error) {
	return suid.DefaultEncoder.Decode(chi.URLParam(r, key))
}
