package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	service "github.com/rapidmidiex/rmx/internal/http"
	"github.com/rapidmidiex/rmx/internal/http/websocket"
	"github.com/rapidmidiex/rmx/internal/jam"

	"github.com/go-chi/chi/v5"
	"github.com/gobwas/ws"
)

type (
	store interface {
		CreateJam(context.Context, jam.Jam) (jam.Jam, error)
		GetJams(context.Context) ([]jam.Jam, error)
		GetJamByID(ctx context.Context, id uuid.UUID) (jam.Jam, error)
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
		if err := s.mux.Decode(w, r, &j); err != nil && err != io.EOF {
			s.mux.Logf("decode: %v\n", err)
			s.mux.Respond(w, r, err, http.StatusBadRequest)
			return
		}

		j.SetDefaults()

		created, err := s.store.CreateJam(r.Context(), j)
		if err != nil {
			s.mux.Logf("createJam: %v\n", err)
			s.mux.Respond(w, r, errors.New("could not create Jam"), http.StatusInternalServerError)
			return
		}

		s.mux.Respond(w, r, created, http.StatusCreated)
	}
}

func (s *jamService) handleGetJam() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// decode uuid from URL
		jamID, err := s.parseUUID(r)

		if err != nil {
			s.mux.Logf("parseUUID: %v\n", err)
			s.mux.Respond(w, r, jamID, http.StatusBadRequest)
			return
		}

		jam, err := s.store.GetJamByID(r.Context(), jamID)
		if err != nil {
			// TODO: Check if error actually is not found or something else
			s.mux.Logf("getJamByID: %v\n", err)
			s.mux.Respond(w, r, err, http.StatusNotFound)
			return
		}

		s.mux.Respond(w, r, jam, http.StatusOK)
	}
}

func (s *jamService) handleListJams() http.HandlerFunc {
	type roomResp struct {
		jam.Jam
		PlayerCount int `json:"playerCount"`
	}
	type response struct {
		Rooms []roomResp `json:"rooms"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		jams, err := s.store.GetJams(r.Context())
		if err != nil {
			s.mux.Logf("getJams: %v", err)
			s.mux.Respond(w, r, "Could not fetch Jams.", http.StatusInternalServerError)
			return
		}
		var resp response
		resp.Rooms = make([]roomResp, 0)
		for _, j := range jams {
			resp.Rooms = append(resp.Rooms, roomResp{
				j,
				s.wsb.ConnCount(j.ID),
			})
		}
		s.mux.Respond(w, r, resp, http.StatusOK)
	}
}

func (s *jamService) handleP2PComms() http.HandlerFunc {
	// FIXME - move to websocket package
	var ErrCapacity = fmt.Errorf("subscriber has reached max capacity")

	return func(w http.ResponseWriter, r *http.Request) {
		jamID, err := s.parseUUID(r)
		if err != nil {
			s.mux.Logf("parseUUID: %v\n", err)
			s.mux.Respond(w, r, jamID, http.StatusBadRequest)
			return
		}

		jamInfo, err := s.store.GetJamByID(r.Context(), jamID)
		if err != nil {
			s.mux.Respond(w, r, err, http.StatusNotFound)
			return
		}

		room, err := s.wsb.GetRoom(jamInfo.ID)
		if err != nil {
			// Create a new Jam room if one does not exist
			if err == websocket.ErrRoomNotFound {
				room = s.newRoom(jamInfo.ID)
			} else {
				s.mux.Logf("getRoom: %v\n", err)
				s.mux.Respond(w, r, err, http.StatusNotFound)
				return
			}
		}

		if room.IsFull() {
			s.mux.Respond(w, r, ErrCapacity, http.StatusServiceUnavailable)
			return
		}

		rwc, _, _, err := ws.UpgradeHTTP(r, w)
		if err != nil {
			// NOTE - this we discovered isn't needed
			s.mux.Logf("upgradeHTTP: %v\n", err)
			s.mux.Respond(w, r, err, http.StatusUpgradeRequired)
			return
		}

		// NOTE - not sure what this actually does,
		// should be coming from database
		u := jam.NewUser("")

		conn := room.NewConn(rwc, u)
		err = room.Subscribe(conn)
		if err != nil {
			s.mux.Logf("subscribe: %v\n", err)
			if err := rwc.Close(); err != nil {
				s.mux.Logf("close: %v\n", err)
				s.mux.Respond(w, r, err, http.StatusInternalServerError)
			}
		}
	}
}

func (s *jamService) routes() {
	s.mux.Route("/api/v1/jam", func(r chi.Router) {
		r.Get("/", s.handleListJams())
		r.Get("/{uuid}", s.handleGetJam())
		r.Post("/", s.handleCreateJam())
	})

	s.mux.Route("/ws/jam", func(r chi.Router) {
		r.Get("/{uuid}", s.handleP2PComms())
	})

}

func (s *jamService) parseUUID(r *http.Request) (uuid.UUID, error) {
	jamID := chi.URLParam(r, "uuid")
	return uuid.Parse(jamID)
}

func (s *jamService) newRoom(jamID uuid.UUID) *websocket.Room[jam.Jam, jam.User] {
	room := websocket.NewRoom[jam.Jam, jam.User](websocket.NewRoomArgs{
		Context:        s.wsb.Context,
		ReadBufferSize: 512,
		ReadTimeout:    defaultTimeout,
		WriteTimeout:   defaultTimeout,
		JamID:          jamID,
	})

	s.wsb.Subscribe(room)
	return room
}

type Option func(*jamService)

func WithBroker(ctx context.Context, cap uint) Option {
	return func(s *jamService) {
		s.wsb = websocket.NewBroker[jam.Jam, jam.User](cap, ctx)
	}
}
