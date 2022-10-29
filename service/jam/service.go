package jam

import (
	"context"
	"errors"
	"log"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"

	h "github.com/hyphengolang/prelude/http"

	"github.com/rog-golang-buddies/rmx/internal/fp"
	"github.com/rog-golang-buddies/rmx/internal/suid"
	ws "github.com/rog-golang-buddies/rmx/internal/websocket"
	websocket "github.com/rog-golang-buddies/rmx/internal/websocket/x"
)

type Service struct {
	m chi.Router
	c *ws.Client

	log  func(s ...any)
	logf func(string, ...any)

	created func(http.ResponseWriter, *http.Request, string)
	respond func(http.ResponseWriter, *http.Request, any, int)
	decode  func(http.ResponseWriter, *http.Request, interface{}) error
}

func (s *Service) routes() {
	// key=suid.SUID <&> value=*websocket.Pool
	var mux sync.Map

	s.m.Route("/api/v1/jam", func(r chi.Router) {
		r.Get("/", s.handleListRooms())
		r.Post("/", s.handleCreateRoom(&mux))
		r.Get("/{uuid}", s.handleGetRoom())
	})

	s.m.Route("/ws/jam", func(r chi.Router) {
		r.Get("/{uuid}", s.handleP2PComms(&mux))
	})

}

func (s *Service) handleP2PComms(mux *sync.Map) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// decode uuid from
		sid, err := s.parseUUID(w, r)
		if err != nil {
			s.respond(w, r, err, http.StatusBadRequest)
			return
		}

		// NOTE this is avoidable, maybe(?)
		value, ok := mux.Load(sid.ShortUUID())
		if !ok {
			s.respond(w, r, "not found", http.StatusNotFound)
			return
		}

		// NOTE this is avoidable, maybe(?)
		pool := (value).(*websocket.Pool)
		conn, close, err := s.upgradeHTTP(w, r, pool)
		if err != nil {
			s.respond(w, r, err, http.StatusUpgradeRequired)
			return
		}

		defer close()
		for {
			msg, err := conn.ReadString()
			if err != nil {
				conn.WriteString(err.Error())
				return
			}

			pool.Broadcast(msg)
		}
	}
}

func (s *Service) handleCreateRoom(mux *sync.Map) http.HandlerFunc {
	type payload struct {
		Capacity uint `json:"capacity"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var pl payload
		if err := s.decode(w, r, &pl); err != nil {
			s.respond(w, r, err, http.StatusBadRequest)
			return
		}

		// NOTE add to Mux/Router/Client whatever it will be called
		sid := suid.NewSUID()
		mux.Store(sid, &websocket.Pool{Capacity: pl.Capacity})

		s.created(w, r, sid.String())
	}
}

func (s *Service) handleGetRoom() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid, err := s.parseUUID(w, r)
		if err != nil {
			s.respond(w, r, err, http.StatusBadRequest)
			return
		}

		// FIXME possible rename
		// method as `Get` is nondescriptive
		p, err := s.c.Get(uid)
		if err != nil {
			s.respond(w, r, err, http.StatusNotFound)
			return
		}

		v := &session{
			ID:    p.ID.ShortUUID(),
			Users: fp.FMap(p.Keys(), func(uid suid.UUID) suid.SUID { return uid.ShortUUID() }),
		}

		s.respond(w, r, v, http.StatusOK)
	}
}

func (s *Service) handleListRooms() http.HandlerFunc {
	type response struct {
		Sessions []session `json:"sessions"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		v := &response{
			Sessions: fp.FMap(s.c.List(), func(p *ws.Pool) session {
				return session{
					ID: p.ID.ShortUUID(),
					Users: fp.FMap(
						p.Keys(),
						func(uid suid.UUID) suid.SUID { return uid.ShortUUID() },
					),
					UserCount: p.Size(),
				}
			}),
		}

		s.respond(w, r, v, http.StatusOK)
	}
}

func (s *Service) upgradeHTTP(w http.ResponseWriter, r *http.Request, pool *websocket.Pool) (conn websocket.Conn, close func(), err error) {
	if pool.IsCap() {
		return nil, nil, errors.New("error: pool has reached capacity")
	}

	if conn, err = websocket.UpgradeHTTP(w, r); err != nil {
		return nil, nil, err
	}

	pool.Append(conn)
	return conn, func() { pool.Remove(conn) }, nil
}

var (
	ErrNoCookie        = errors.New("api: cookie not found")
	ErrSessionNotFound = errors.New("api: session not found")
	ErrSessionExists   = errors.New("api: session already exists")
)

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.m.ServeHTTP(w, r) }

func NewService(ctx context.Context, r chi.Router) *Service {
	s := &Service{
		r,
		ws.DefaultClient, log.Print, log.Printf, h.Created, h.Respond, h.Decode,
	}
	s.routes()
	return s
}

func (s *Service) parseUUID(w http.ResponseWriter, r *http.Request) (suid.UUID, error) {
	return suid.ParseString(chi.URLParam(r, "uuid"))
}

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
