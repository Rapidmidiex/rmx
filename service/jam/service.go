package jam

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"

	h "github.com/hyphengolang/prelude/http"

	"github.com/rog-golang-buddies/rmx/internal"
	"github.com/rog-golang-buddies/rmx/internal/fp"
	"github.com/rog-golang-buddies/rmx/internal/suid"
	ws "github.com/rog-golang-buddies/rmx/internal/websocket"
	w2 "github.com/rog-golang-buddies/rmx/internal/websocket/x"
)

type Pool struct {
	mu sync.Mutex
	m  map[w2.Conn]bool
}

func (p *Pool) BroadcastString(s string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for c := range p.m {
		if err := c.WriteString(s); err != nil {
			return err
		}
	}

	return nil
}

func (p *Pool) remove(c w2.Conn) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := c.Close(); err != nil {
		return err
	}
	delete(p.m, c)
	return nil
}

func (s *Service) routes() {
	s.m.Route("/api/v1/jam", func(r chi.Router) {
		r.Get("/", s.handleListRooms())
		r.Post("/", s.handleCreateRoom())
		r.Get("/{uuid}", s.handleGetRoom())
	})

	s.m.Route("/ws", func(r chi.Router) {

		pool := Pool{m: make(map[w2.Conn]bool)}

		r.Route("/echo", func(r chi.Router) {
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				conn, err := w2.UpgradeHTTP(w, r)
				if err != nil {
					s.respond(w, r, err, http.StatusUpgradeRequired)
					return
				}

				pool.m[conn] = true

				defer pool.remove(conn)
				for {
					msg, err := conn.ReadString()
					if err != nil {
						return
					}
					// s.log("connection added")
					if err := pool.BroadcastString(msg); err != nil {
						return
					}
				}
			})
		})

		r.Route("/jam", func(r chi.Router) {
			r = r.With(s.connectionPool(nil), s.upgradeHTTP(1024, 1024))
			r.Get("/{uuid}", s.handleP2PComms())
		})
	})

}

func (s *Service) handleP2PComms() http.HandlerFunc {
	// FIXME we will change this as I know this hasn't been
	// was just my way of getting things working, not yet
	// full agreement with this.
	type response[T any] struct {
		Typ     internal.MsgTyp `json:"type"`
		Payload T               `json:"payload"`
	}

	type join struct {
		ID        suid.SUID `json:"id"`
		SessionID suid.SUID `json:"sessionId"`
	}

	type leave struct {
		ID        suid.SUID `json:"id"`
		SessionID suid.SUID `json:"sessionId"`
		Error     any       `json:"err"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		c := r.Context().Value(internal.UpgradeKey).(*ws.Conn)

		defer func() {
			// FIXME send error when Leaving session pool
			c.SendMessage(response[leave]{
				Typ:     internal.Leave,
				Payload: leave{ID: c.ID.ShortUUID(), SessionID: c.Pool().ID.ShortUUID()},
			})

			c.Close()
		}()

		if err := c.SendMessage(response[join]{
			Typ:     internal.Join,
			Payload: join{ID: c.ID.ShortUUID(), SessionID: c.Pool().ID.ShortUUID()},
		}); err != nil {
			s.log(err)
			return
		}

		// TODO could the API be adjusted such that
		// this for-loop only needs to read and
		// never touch the code for writing
		for {
			var msg response[json.RawMessage]
			if err := c.ReadJSON(&msg); err != nil {
				s.log(err)
				return
			}

			// TODO here the message will be passed off to a different handler
			// via a go routine*
			if err := c.SendMessage(response[int]{Typ: internal.Message, Payload: 10}); err != nil {
				s.log(err)
				return
			}
		}
	}
}

func (s *Service) handleCreateRoom() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid, err := s.c.NewPool(4)
		if err != nil {
			s.respond(w, r, err, http.StatusInternalServerError)
			return
		}

		v := &session{ID: suid.FromUUID(uid)}

		s.respond(w, r, v, http.StatusOK)
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
