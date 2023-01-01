package v2

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gobwas/ws"
	"github.com/hyphengolang/prelude/types/suid"
	"github.com/rog-golang-buddies/rmx/internal/fp"
	"github.com/rog-golang-buddies/rmx/internal/websocket"
	"github.com/rog-golang-buddies/rmx/pkg/service"
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
	service.Service
}

func NewService(ctx context.Context, mux chi.Router) *Service {
	s := &Service{service.New(ctx, mux)}
	s.routes()
	return s
}

const (
	defaultTimeout = time.Second * 10
)

type User struct {
	id       suid.UUID
	Username string `json:"username"`
}

func (u *User) fillDefaults() {
	u.id = suid.NewUUID()
	if strings.TrimSpace(u.Username) == "" {
		u.Username = u.id.String()
	}
}

type Jam struct {
	id       suid.UUID
	owner    *User
	Name     string `json:"name,omitempty"`
	Capacity uint   `json:"capacity,omitempty"`
	BPM      uint   `json:"bpm,omitempty"`
}

func (j *Jam) fillDefaults() {
	j.id = suid.NewUUID()
	if j.owner == nil {
		j.owner = &User{j.id, j.Name}
	}
	if strings.TrimSpace(j.Name) == "" {
		j.Name = j.id.ShortUUID().String()
	}
	if j.Capacity == 0 {
		j.Capacity = 10
	}
	if j.BPM == 0 {
		j.BPM = 80
	}
}

func (s *Service) handleCreateJamRoom(b *websocket.Broker[Jam, User]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var j Jam
		if err := s.Decode(w, r, &j); err != nil {
			s.Respond(w, r, err, http.StatusBadRequest)
			return
		}

		// fill out empty fields with default value.
		j.fillDefaults()

		// create a new Subscriber
		sub := websocket.NewSubscriber[Jam, User](
			b.Context,
			j.Capacity,
			512,
			defaultTimeout,
			defaultTimeout,
			&j,
		)

		// connect the Subscriber
		b.Subscribe(sub)

		s.Created(w, r, sub.GetID().ShortUUID().String())
	}
}

func (s *Service) handleGetRoomData(b *websocket.Broker[Jam, User]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// decode uuid from URL
		sid, err := s.parseUUID(r)
		if err != nil {
			s.Respond(w, r, sid, http.StatusBadRequest)
			return
		}

		sub, err := b.GetSubscriber(sid)
		if err != nil {
			s.Respond(w, r, err, http.StatusNotFound)
			return
		}

		s.Respond(w, r, sub.Info, http.StatusOK)
	}
}

func (s *Service) handleGetRoomUsers(b *websocket.Broker[Jam, User]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// decode uuid from URL
		sid, err := s.parseUUID(r)
		if err != nil {
			s.Respond(w, r, sid, http.StatusBadRequest)
			return
		}

		sub, err := b.GetSubscriber(sid)
		if err != nil {
			s.Respond(w, r, err, http.StatusNotFound)
			return
		}

		conns := sub.ListConns()

		connsInfo := fp.FMap(conns, func(c *websocket.Conn[User]) User {
			return *c.Info
		})

		s.Respond(w, r, connsInfo, http.StatusOK)
	}
}

func (s *Service) handleListRooms(b *websocket.Broker[Jam, User]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		subs := b.ListSubscribers()
		subsInfo := fp.FMap(subs, func(s *websocket.Subscriber[Jam, User]) Jam {
			return *s.Info
		})

		s.Respond(w, r, subsInfo, http.StatusOK)
	}
}

func (s *Service) handleP2PComms(b *websocket.Broker[Jam, User]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// decode uuid from URL
		sid, err := s.parseUUID(r)
		if err != nil {
			s.Respond(w, r, sid, http.StatusBadRequest)
			return
		}

		sub, err := b.GetSubscriber(sid)
		if err != nil {
			s.Respond(w, r, err, http.StatusNotFound)
			return
		}

		if err := errors.New("subscriber has reached max capacity"); sub.IsFull() {
			s.Respond(w, r, err, http.StatusServiceUnavailable)
			return
		}

		rwc, _, _, err := ws.UpgradeHTTP(r, w)
		if err != nil {
			s.Respond(w, r, err, http.StatusUpgradeRequired)
			return
		}

		var u User
		u.fillDefaults()

		conn := sub.NewConn(rwc, &u)
		sub.Subscribe(conn)
	}
}

func (s *Service) routes() {
	broker := websocket.NewBroker[Jam, User](10, context.Background())

	s.Route("/api/v1/jam", func(r chi.Router) {
		r.Get("/", s.handleListRooms(broker))
		r.Get("/{uuid}", s.handleGetRoomData(broker))
		r.Get("/{uuid}/users", s.handleGetRoomUsers(broker))
		r.Post("/", s.handleCreateJamRoom(broker))
	})

	s.Route("/ws/jam", func(r chi.Router) {
		r.Get("/{uuid}", s.handleP2PComms(broker))
	})

}

func (s *Service) parseUUID(r *http.Request) (suid.UUID, error) {
	return suid.ParseString(chi.URLParam(r, "uuid"))
}
