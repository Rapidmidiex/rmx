package service

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	service "github.com/rapidmidiex/rmx/pkg/http"

	"github.com/hyphengolang/prelude/http/websocket"

	"github.com/hyphengolang/prelude/types/suid"
)

// Jam Service Endpoints
//
// Create a new jam session.
//
//	POST /api/v1/jam
//
// List all jam sessions metadata.
//
//	GET /api/v1/jam
//
// Get a jam sessions metadata.
//
//	GET /api/v1/jam/{uuid}
//
// Connect to jam session.
//
//	GET /ws/jam/{uuid}
type Service struct {
	mux service.Service

	// c *ws.Client
}

func (s *Service) routes() {
	// NOTE this map is temporary
	//  map[suid.SUID]*websocket.Pool
	var broker = &mux{
		mp: make(map[suid.UUID]muxEntry),
	}

	s.mux.Route("/api/v1/jam", func(r chi.Router) {
		// r.Get("/", s.handleListRooms())
		r.Post("/", s.handleCreateJamRoom(broker))
		// r.Get("/{uuid}", s.handleGetRoomData(mux))
	})

	s.mux.Route("/ws/jam", func(r chi.Router) {
		r.Get("/{uuid}", s.handleP2PComms(broker))
	})

}

func (s *Service) handleP2PComms(mux *mux) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// decode uuid from
		sid, err := s.parseUUID(w, r)
		if err != nil {
			s.mux.Respond(w, r, err, http.StatusBadRequest)
			return
		}

		pool, err := mux.Load(sid)
		if err != nil {
			s.mux.Respond(w, r, err, http.StatusNotFound)
			return
		}

		if err := errors.New("pool has reached max capacity"); pool.IsFull() {
			s.mux.Respond(w, r, err, http.StatusServiceUnavailable)
			return
		}

		rwc, err := websocket.UpgradeHTTP(w, r)
		if err != nil {
			s.mux.Respond(w, r, err, http.StatusUpgradeRequired)
			return
		}

		pool.ListenAndServe(rwc)
	}
}

func (s *Service) handleCreateJamRoom(mux *mux) http.HandlerFunc {
	type payload struct {
		Capacity uint `json:"capacity"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var pl payload
		if err := s.mux.Decode(w, r, &pl); err != nil {
			s.mux.Respond(w, r, err, http.StatusBadRequest)
			return
		}

		pool := &websocket.Pool{
			Capacity:       pl.Capacity,
			ReadBufferSize: 512,
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
		}

		e := muxEntry{suid.NewUUID(), pool}

		mux.Store(e)
		s.mux.Created(w, r, e.String())
	}
}

// func (s *Service) handleGetRoomData(mux *mux) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		uid, err := s.parseUUID(w, r)
// 		if err != nil {
// 			s.mux.Respond(w, r, err, http.StatusBadRequest)
// 			return
// 		}

// 		// FIXME possible rename
// 		// method as `Get` is nondescriptive
// 		p, err := s.c.Get(uid)
// 		if err != nil {
// 			s.mux.Respond(w, r, err, http.StatusNotFound)
// 			return
// 		}

// 		v := &Session{
// 			ID:    p.ID.ShortUUID(),
// 			Users: fp.FMap(p.Keys(), func(uid suid.UUID) suid.SUID { return uid.ShortUUID() }),
// 		}

// 		s.mux.Respond(w, r, v, http.StatusOK)
// 	}
// }

// func (s *Service) handleListRooms() http.HandlerFunc {
// 	type response struct {
// 		Sessions []Session `json:"sessions"`
// 	}

// 	return func(w http.ResponseWriter, r *http.Request) {
// 		v := &response{
// 			Sessions: fp.FMap(s.c.List(), func(p *ws.Pool) Session {
// 				return Session{
// 					ID: p.ID.ShortUUID(),
// 					Users: fp.FMap(
// 						p.Keys(),
// 						func(uid suid.UUID) suid.SUID { return uid.ShortUUID() },
// 					),
// 					UserCount: p.Size(),
// 				}
// 			}),
// 		}

// 		s.mux.Respond(w, r, v, http.StatusOK)
// 	}
// }

// func (s *Service) upgradeHTTP(w http.ResponseWriter, r *http.Request, pool *websocket.Pool) (conn websocket.Conn, close func(), err error) {
// 	if pool.IsCap() {
// 		return nil, nil, errors.New("error: pool has reached capacity")
// 	}

// 	if conn, err = websocket.UpgradeHTTP(w, r); err != nil {
// 		return nil, nil, err
// 	}

// 	pool.Append(conn)
// 	return conn, func() { pool.Remove(conn) }, nil
// }

var (
	ErrNoCookie        = errors.New("api: cookie not found")
	ErrSessionNotFound = errors.New("api: session not found")
	ErrSessionExists   = errors.New("api: session already exists")
)

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func NewService(mux chi.Router) http.Handler {
	s := &Service{mux: service.New(service.WithRouter(mux))}

	s.routes()
	return s
}

func (s *Service) parseUUID(w http.ResponseWriter, r *http.Request) (suid.UUID, error) {
	return suid.ParseString(chi.URLParam(r, "uuid"))
}

type Jam struct {
	Name string `json:"name"`
	BPM  int    `json:"bpm"`
}

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
