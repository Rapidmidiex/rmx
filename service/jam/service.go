package jam

import (
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	h "github.com/hyphengolang/prelude/http"

	"github.com/rog-golang-buddies/rmx/internal/suid"
	ws "github.com/rog-golang-buddies/rmx/internal/websocket"
)

type contextKey string

var (
	roomKey    = contextKey("rmx-fetch-pool")
	upgradeKey = contextKey("rmx-upgrade-http")
)

func chain(hf http.HandlerFunc, mw ...h.MiddleWare) http.HandlerFunc { return h.Chain(hf, mw...) }

var (
	ErrNoCookie        = errors.New("api: cookie not found")
	ErrSessionNotFound = errors.New("api: session not found")
	ErrSessionExists   = errors.New("api: session already exists")
)

type jam struct {
	Name string `json:"name"`
	BPM  int    `json:"bpm"`
	ws.Pool
}

type session struct {
	ID    suid.SUID   `json:"id"`
	Name  string      `json:"name,omitempty"`
	Users []suid.SUID `json:"users,omitempty"`
	/* Not really required */
	UserCount int `json:"userCount"`
}

type User struct {
	ID   suid.SUID `json:"id"`
	Name string    `json:"name,omitempty"`
	/* More fields can belong here */
}

type Service struct {
	m chi.Router
	l *log.Logger

	c *ws.Client
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.m.ServeHTTP(w, r) }

func NewService(r chi.Router) *Service {
	s := &Service{r, log.Default(), ws.DefaultClient}
	s.routes()
	return s
}

func DefaultService() *Service {
	s := &Service{chi.NewMux(), log.Default(), ws.DefaultClient}
	s.routes()
	return s
}

func (s *Service) respond(w http.ResponseWriter, r *http.Request, data any, status int) {
	h.Respond(w, r, data, status)
}

func (s *Service) decode(w http.ResponseWriter, r *http.Request, data interface{}) error {
	return h.Decode(w, r, data)
}

func (s *Service) parseUUID(w http.ResponseWriter, r *http.Request) (suid.UUID, error) {
	return suid.ParseString(chi.URLParam(r, "uuid"))
}

func (s *Service) routes() {
	s.m.Route("/api/v1/jam", func(r chi.Router) {
		r.Get("/", s.handleListRooms())
		r.Post("/", s.handleCreateRoom())
		r.Get("/{uuid}", s.handleGetRoom())

		r.Get("/ping", s.handlePing)
	})

	// s.m.Get("/ws/jam/{uuid}", chain(s.handleP2PComms(), s.upgradeHTTP(1024, 1024), s.connectionPool(nil)))
	s.m.Route("/ws/jam", func(r chi.Router) {
		r = r.With(s.connectionPool(nil), s.upgradeHTTP(1024, 1024))
		r.Get("/{uuid}", s.handleP2PComms())
	})
}

func (s *Service) handlePing(w http.ResponseWriter, r *http.Request) {
	s.respond(w, r, nil, http.StatusNoContent)
}
