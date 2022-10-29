package jam

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"

	h "github.com/hyphengolang/prelude/http"

	"github.com/rog-golang-buddies/rmx/internal"
	"github.com/rog-golang-buddies/rmx/internal/fp"
	"github.com/rog-golang-buddies/rmx/internal/suid"
	ws "github.com/rog-golang-buddies/rmx/internal/websocket"
	w2 "github.com/rog-golang-buddies/rmx/internal/websocket/x"
)

func (s *Service) routes() {
	s.m.Route("/api/v1/jam", func(r chi.Router) {
		r.Get("/", s.handleListRooms())
		r.Post("/", s.handleCreateRoom())
		r.Get("/{uuid}", s.handleGetRoom())
	})

	// create a single Pool
	pool := &w2.Pool{Capacity: 2}

	s.m.Route("/ws/jam", func(r chi.Router) {
		r.Get("/{uuid}", s.handleP2PComms(pool))
	})

}

func (s *Service) handleP2PComms(pool *w2.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if pool.IsCap() {
			s.respond(w, r, "error: pool has reached capacity", http.StatusUpgradeRequired)
			return
		}

		conn, err := w2.UpgradeHTTP(w, r)
		if err != nil {
			s.respond(w, r, err, http.StatusUpgradeRequired)
			return
		}

		pool.Append(conn)
		defer pool.Remove(conn)
		for {
			msg, err := conn.ReadString()
			if err != nil {
				return
			}

			pool.Broadcast(msg)
		}
	}
}

func (s *Service) handleCreateRoom() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// uid, err := s.c.NewPool(4)
		// if err != nil {
		// 	s.respond(w, r, err, http.StatusInternalServerError)
		// 	return
		// }

		// v := &session{ID: suid.FromUUID(uid)}

		// s.respond(w, r, v, http.StatusOK)

		s.respond(w, r, nil, http.StatusNotImplemented)
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

// needs to be moved into websocket package as its middleware
func (s *Service) connectionPool(p *ws.Pool) func(f http.Handler) http.Handler {
	return func(f http.Handler) http.Handler {
		var fn func(w http.ResponseWriter, r *http.Request)
		if p != nil {
			fn = func(w http.ResponseWriter, r *http.Request) {
				f.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), internal.RoomKey, p)))
			}
		} else {
			fn = func(w http.ResponseWriter, r *http.Request) {
				uid, err := s.parseUUID(w, r)
				if err != nil {
					s.respond(w, r, err, http.StatusBadRequest)
					return
				}

				p, err := s.c.Get(uid)
				if err != nil {
					s.respond(w, r, err, http.StatusNotFound)
					return
				}

				r = r.WithContext(context.WithValue(r.Context(), internal.RoomKey, p))
				f.ServeHTTP(w, r)
			}
		}

		return http.HandlerFunc(fn)
	}
}

// needs to be moved into websocket package as its middleware
func (s *Service) upgradeHTTP(readBuf, writeBuf int) func(f http.Handler) http.Handler {
	return func(f http.Handler) http.Handler {
		u := &websocket.Upgrader{
			ReadBufferSize:  readBuf,
			WriteBufferSize: writeBuf,
			CheckOrigin:     func(r *http.Request) bool { return true },
		}

		fn := func(w http.ResponseWriter, r *http.Request) {
			p, _ := r.Context().Value(internal.RoomKey).(*ws.Pool)
			if p.Size() == p.MaxConn {
				s.respond(w, r, ws.ErrMaxConn, http.StatusUnauthorized)
				return
			}

			c, err := p.NewConn(w, r, u)
			if err != nil {
				s.respond(w, r, err, http.StatusInternalServerError)
				return
			}

			r = r.WithContext(context.WithValue(r.Context(), internal.UpgradeKey, c))
			f.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}

var (
	ErrNoCookie        = errors.New("api: cookie not found")
	ErrSessionNotFound = errors.New("api: session not found")
	ErrSessionExists   = errors.New("api: session already exists")
)

type Service struct {
	m chi.Router
	c *ws.Client

	log  func(s ...any)
	logf func(string, ...any)

	respond func(http.ResponseWriter, *http.Request, any, int)
	decode  func(http.ResponseWriter, *http.Request, interface{}) error
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.m.ServeHTTP(w, r) }

func NewService(ctx context.Context, r chi.Router) *Service {
	s := &Service{
		r,
		ws.DefaultClient, log.Print, log.Printf, h.Respond, h.Decode,
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
