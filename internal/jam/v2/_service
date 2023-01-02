package v2

import (
	"context"
	"fmt"
	"net/http"
	"time"

	srv "github.com/rog-golang-buddies/rmx/internal/http"
	ws "github.com/rog-golang-buddies/rmx/internal/http/websocket"
	intern "github.com/rog-golang-buddies/rmx/internal/jam-sessions"

	"github.com/go-chi/chi/v5"
	gobwas "github.com/gobwas/ws"
	"github.com/hyphengolang/prelude/types/suid"
)

const (
	defaultTimeout = time.Second * 10
)

type jamService struct {
	mux srv.Service

	wsb *ws.Broker[intern.Jam, intern.User]
}

func (s *jamService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func NewService(ctx context.Context, cap uint) http.Handler {
	broker := ws.NewBroker[intern.Jam, intern.User](ctx, cap)

	s := &jamService{
		mux: srv.New(),
		wsb: broker,
	}

	s.routes()
	return s
}

func (s *jamService) handleCreateJamRoom() http.HandlerFunc {
	// NOTE - should be an intimidate Jam type before it is converted to a domain type
	return func(w http.ResponseWriter, r *http.Request) {
		var j intern.Jam
		if err := s.mux.Decode(w, r, &j); err != nil {
			s.mux.Respond(w, r, err, http.StatusBadRequest)
			return
		}

		sub := s.newSubscriber(&j)
		s.mux.Created(w, r, sub.GetID().ShortUUID().String())
	}
}

func (s *jamService) handleGetRoomData() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.mux.Respond(w, r, nil, http.StatusNotImplemented)
	}
}

func (s *jamService) handleListRooms() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.mux.Respond(w, r, nil, http.StatusNotImplemented)
	}
}

func (s *jamService) handleP2PComms() http.HandlerFunc {
	// FIXME - move to websocket package
	var ErrCapacity = fmt.Errorf("subscriber has reached max capacity")

	return func(w http.ResponseWriter, r *http.Request) {
		// decode uuid from URL
		sid, err := s.parseUUID(w, r)
		if err != nil {
			s.mux.Respond(w, r, sid, http.StatusBadRequest)
			return
		}

		sub, err := s.wsb.GetSubscriber(sid)
		if err != nil {
			s.mux.Respond(w, r, err, http.StatusNotFound)
			return
		}

		if sub.IsFull() {
			s.mux.Respond(w, r, ErrCapacity, http.StatusServiceUnavailable)
			return
		}

		rwc, _, _, err := gobwas.UpgradeHTTP(r, w)
		if err != nil {
			// NOTE - this we discovered isn't needed
			s.mux.Respond(w, r, err, http.StatusUpgradeRequired)
			return
		}

		// NOTE - not sure what this actually does,
		// should be coming from database
		u := intern.NewUser("")

		conn := sub.NewConn(rwc, u)
		sub.Subscribe(conn)
	}
}

func (s *jamService) routes() {
	s.mux.Route("/api/v1/jam", func(r chi.Router) {
		r.Get("/", s.handleListRooms())
		r.Get("/{uuid}", s.handleGetRoomData())
		r.Post("/", s.handleCreateJamRoom())
	})

	s.mux.Route("/ws/jam", func(r chi.Router) {
		r.Get("/{uuid}", s.handleP2PComms())
	})

}

func (s *jamService) parseUUID(w http.ResponseWriter, r *http.Request) (suid.UUID, error) {
	return suid.ParseString(chi.URLParam(r, "uuid"))
}

func (s *jamService) newSubscriber(j *intern.Jam) *ws.Subscriber[intern.Jam, intern.User] {
	sub := ws.NewSubscriber[intern.Jam, intern.User](
		s.wsb.Context, j.Capacity, 512, defaultTimeout, defaultTimeout, j,
	)

	s.wsb.Subscribe(sub)
	return sub
}
