package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	service "github.com/rapidmidiex/rmx/internal/http"
	"github.com/rapidmidiex/rmx/internal/http/websocket"
	"github.com/rapidmidiex/rmx/internal/jam"

	"github.com/go-chi/chi/v5"
	"github.com/gobwas/ws"
	"github.com/hyphengolang/prelude/types/suid"
	"github.com/rapidmidiex/rmx/internal/fp"
)

type (
	store interface {
		CreateJam(context.Context, jam.Jam) error
	}

	jamService struct {
		mux service.Service

		wsb *websocket.Broker[jam.Jam, jam.User]

		store store
	}
)

func (s *jamService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func NewService(ctx context.Context, store store, opts ...Option) http.Handler {
	s := jamService{
		mux:   service.New(),
		store: store,
	}

	for _, opt := range opts {
		opt(&s)
	}

	if s.wsb == nil {
		s.wsb = websocket.NewBroker[jam.Jam, jam.User](10, ctx)
	}

	s.routes()
	return &s
}

const (
	defaultTimeout = time.Second * 10
)

func (s *jamService) handleCreateJam() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var j jam.Jam
		if err := s.mux.Decode(w, r, &j); err != nil {
			s.mux.Respond(w, r, err, http.StatusBadRequest)
			return
		}
		err := s.store.CreateJam(r.Context(), j)
		if err != nil {
			s.mux.Respond(w, r, errors.New("could not create Jam"), http.StatusInternalServerError)
			return
		}

		s.mux.Respond(w, r, j, http.StatusCreated)
	}
}

func (s *jamService) handleCreateJamRoom() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var j jam.Jam
		if err := s.mux.Decode(w, r, &j); err != nil {
			s.mux.Respond(w, r, err, http.StatusBadRequest)
			return
		}

		// Discuss: The client should not be added to the the Jam Room on this REST endpoint
		// Only the Jam resource should be created. Then the client should use the Jam ID to connect
		// to the Jam Room (if they wish).
		sub := s.newSession(&j)
		s.mux.Created(w, r, sub.GetID().ShortUUID().String())
	}
}

func (s *jamService) handleGetRoomData() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// decode uuid from URL
		sid, err := s.parseUUID(r)
		if err != nil {
			s.mux.Respond(w, r, sid, http.StatusBadRequest)
			return
		}

		ses, err := s.wsb.GetSession(sid)
		if err != nil {
			s.mux.Respond(w, r, err, http.StatusNotFound)
			return
		}

		s.mux.Respond(w, r, ses.Info, http.StatusOK)
	}
}

func (s *jamService) handleGetRoomUsers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// decode uuid from URL
		sid, err := s.parseUUID(r)
		if err != nil {
			s.mux.Respond(w, r, sid, http.StatusBadRequest)
			return
		}

		ses, err := s.wsb.GetSession(sid)
		if err != nil {
			s.mux.Respond(w, r, err, http.StatusNotFound)
			return
		}

		// NOTE - subject to change
		conns := ses.ListConns()
		connsInfo := fp.FMap(conns, func(c *websocket.Conn[jam.User]) jam.User {
			return *c.Info
		})

		s.mux.Respond(w, r, connsInfo, http.StatusOK)
	}
}

func (s *jamService) handleListRooms() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// NOTE - subject to change
		ss := s.wsb.ListSessions()
		subsInfo := fp.FMap(ss, func(s *websocket.Session[jam.Jam, jam.User]) jam.Jam {
			return *s.Info
		})

		s.mux.Respond(w, r, subsInfo, http.StatusOK)
	}
}

func (s *jamService) handleP2PComms() http.HandlerFunc {
	// FIXME - move to websocket package
	var ErrCapacity = fmt.Errorf("subscriber has reached max capacity")

	return func(w http.ResponseWriter, r *http.Request) {
		sid, err := s.parseUUID(r)
		if err != nil {
			s.mux.Respond(w, r, sid, http.StatusBadRequest)
			return
		}

		ses, err := s.wsb.GetSession(sid)
		if err != nil {
			s.mux.Respond(w, r, err, http.StatusNotFound)
			return
		}

		if ses.IsFull() {
			s.mux.Respond(w, r, ErrCapacity, http.StatusServiceUnavailable)
			return
		}

		rwc, _, _, err := ws.UpgradeHTTP(r, w)
		if err != nil {
			// NOTE - this we discovered isn't needed
			s.mux.Respond(w, r, err, http.StatusUpgradeRequired)
			return
		}

		// NOTE - not sure what this actually does,
		// should be coming from database
		u := jam.NewUser("")

		conn := ses.NewConn(rwc, u)
		ses.Subscribe(conn)
	}
}

func (s *jamService) routes() {
	s.mux.Route("/api/v1/jam", func(r chi.Router) {
		r.Get("/", s.handleListRooms())
		r.Get("/{uuid}", s.handleGetRoomData())
		r.Get("/{uuid}/users", s.handleGetRoomUsers())
		r.Post("/", s.handleCreateJam())
	})

	s.mux.Route("/ws/jam", func(r chi.Router) {
		r.Get("/{uuid}", s.handleP2PComms())
	})

}

func (s *jamService) parseUUID(r *http.Request) (suid.UUID, error) {
	return suid.ParseString(chi.URLParam(r, "uuid"))
}

func (s *jamService) newSession(j *jam.Jam) *websocket.Session[jam.Jam, jam.User] {
	ses := websocket.NewSession[jam.Jam, jam.User](
		s.wsb.Context, j.Capacity, 512, defaultTimeout, defaultTimeout, j,
	)

	s.wsb.Subscribe(ses)
	return ses
}

type Option func(*jamService)

func WithBroker(ctx context.Context, cap uint) Option {
	return func(s *jamService) {
		s.wsb = websocket.NewBroker[jam.Jam, jam.User](cap, ctx)
	}
}
