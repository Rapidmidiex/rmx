package service

import (
	"encoding/json"
	"net/http"

	rmx "github.com/rog-golang-buddies/rapidmidiex/internal"
	"github.com/rog-golang-buddies/rapidmidiex/internal/suid"
	ws "github.com/rog-golang-buddies/rapidmidiex/internal/websocket"
)

func (s Service) handleP2PComms() http.HandlerFunc {
	type response[T any] struct {
		Typ     rmx.MessageTyp `json:"type"`
		Payload T              `json:"payload"`
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
		c := r.Context().Value(upgradeKey).(*ws.Conn)
		defer func() {
			// ! send error when Leaving session pool
			c.SendMessage(response[leave]{
				Typ:     rmx.Leave,
				Payload: leave{ID: c.ID.ShortUUID(), SessionID: c.Pool().ID.ShortUUID()},
			})

			c.Close()
		}()

		if err := c.SendMessage(response[join]{
			Typ:     rmx.Join,
			Payload: join{ID: c.ID.ShortUUID(), SessionID: c.Pool().ID.ShortUUID()},
		}); err != nil {
			s.l.Println(err)
			return
		}

		// ? could the API be adjusted such that
		// ? this for-loop only needs to read and
		// ? never touch the code for writing
		for {
			var msg response[json.RawMessage]
			if err := c.ReadJSON(&msg); err != nil {
				s.l.Println(err)
				return
			}

			// * here the message will be passed off to a different handler
			// * via a go routine*
			if err := c.SendMessage(response[int]{Typ: rmx.Message, Payload: 10}); err != nil {
				s.l.Println(err)
				return
			}
		}
	}
}

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
