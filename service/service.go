package service

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"

	h "github.com/hyphengolang/prelude/http"

	"github.com/rog-golang-buddies/rapidmidiex/internal/suid"
	ws "github.com/rog-golang-buddies/rapidmidiex/internal/websocket"
)

type contextKey string

var (
	roomKey    = contextKey("rmx-fetch-pool")
	upgradeKey = contextKey("rmx-upgrade-http")
)

func chain(hf http.HandlerFunc, mw ...h.MiddleWare) http.HandlerFunc { return h.Chain(hf, mw...) }

type Session struct {
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
	r chi.Router
	l *log.Logger

	c *ws.Client
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.r.ServeHTTP(w, r) }

func New(r chi.Router) *Service {
	s := &Service{r, log.Default(), ws.DefaultClient}
	s.routes()
	return s
}

func (s *Service) respond(w http.ResponseWriter, r *http.Request, data any, status int) {
	h.Respond(w, r, data, status)
}

// ! deprecated
// func (s *Service) decode(w http.ResponseWriter, r *http.Request, data interface{}) error {	return h.Decode(w, r, data) }
// func (s *Service) fileServer(prefix string, dirname string) http.Handler {	return h.FileServer(prefix, dirname) }

func (s *Service) parseUUID(w http.ResponseWriter, r *http.Request) (suid.UUID, error) {
	return suid.ParseString(chi.URLParam(r, "id"))
}

func (s *Service) routes() {
	// middleware
	s.r.Use(middleware.Logger)

	// temporary static files
	// s.r.Handle("/assets/*", s.fileServer("/assets/", "assets"))
	// s.r.Get("/", s.indexHTML("ui/www/index.html"))
	// s.r.Get("/play/{id}", s.jamSessionHTML("ui/www/play.html"))

	// REST v1
	s.r.Get("/api/v1/jam", s.handleListRooms())
	s.r.Post("/api/v1/jam", s.handleCreateRoom())
	s.r.Get("/api/v1/jam/{id}", s.handleGetRoom())

	s.r.Get("/ping", s.handlePing)

	// Websocket
	s.r.Get("/ws/jam/{id}", chain(s.handleP2PComms(), s.upgradeHTTP(1024, 1024), s.connectionPool(nil)))

	s.r.Get("/ws/ping", chain(s.handleEcho(), s.upgradeHTTP(1024, 1024), s.connectionPool(ws.DefaultPool())))
}
