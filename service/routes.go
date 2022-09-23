package service

import (
	"net/http"

	rmx "github.com/rog-golang-buddies/rapidmidiex/internal"
	"github.com/rog-golang-buddies/rapidmidiex/internal/suid"
	ws "github.com/rog-golang-buddies/rapidmidiex/internal/websocket"
)

func (s Service) handleCreateRoom() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid, err := s.c.NewPool()
		if err != nil {
			s.respond(w, r, err, http.StatusInternalServerError)
			return
		}

		v := Session{ID: suid.FromUUID(uid)}

		s.respond(w, r, v, http.StatusOK)
	}
}

func (s Service) handleGetRoom() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid, err := s.parseUUID(w, r)
		if err != nil {
			s.respond(w, r, err, http.StatusBadRequest)
			return
		}

		// ! rename method as `Get` is nondescriptive
		p, err := s.c.Get(uid)
		if err != nil {
			s.respond(w, r, err, http.StatusNotFound)
			return
		}

		v := &Session{
			ID:    p.ID.ShortUUID(),
			Users: rmx.FMap(p.Keys(), func(uid suid.UUID) suid.SUID { return uid.ShortUUID() }),
		}

		s.respond(w, r, v, http.StatusOK)
	}
}

func (s Service) handleListRooms() http.HandlerFunc {
	type response struct {
		Sessions []Session `json:"sessions"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		v := &response{
			Sessions: rmx.FMap(s.c.List(), func(p *ws.Pool) Session {
				return Session{
					ID:        p.ID.ShortUUID(),
					Users:     rmx.FMap(p.Keys(), func(uid suid.UUID) suid.SUID { return uid.ShortUUID() }),
					UserCount: p.Size(),
				}
			}),
		}

		s.respond(w, r, v, http.StatusOK)
	}
}

func (s Service) handlePing(w http.ResponseWriter, r *http.Request) {
	s.respond(w, r, nil, http.StatusNoContent)
}
