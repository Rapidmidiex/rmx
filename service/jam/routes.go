package jam

import (
	"net/http"

	"context"
	"encoding/json"

	"github.com/gorilla/websocket"
	rmx "github.com/rog-golang-buddies/rmx/internal"
	"github.com/rog-golang-buddies/rmx/internal/fp"
	"github.com/rog-golang-buddies/rmx/internal/suid"
	ws "github.com/rog-golang-buddies/rmx/internal/websocket"

	t "github.com/hyphengolang/prelude/template"
)

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

		// ! rename method as `Get` is nondescriptive
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
					ID:        p.ID.ShortUUID(),
					Users:     fp.FMap(p.Keys(), func(uid suid.UUID) suid.SUID { return uid.ShortUUID() }),
					UserCount: p.Size(),
				}
			}),
		}

		s.respond(w, r, v, http.StatusOK)
	}
}

func (s *Service) handlePing(w http.ResponseWriter, r *http.Request) {
	s.respond(w, r, nil, http.StatusNoContent)
}

func (s *Service) connectionPool(p *ws.Pool) func(f http.HandlerFunc) http.HandlerFunc {
	return func(f http.HandlerFunc) http.HandlerFunc {
		if p != nil {
			return func(w http.ResponseWriter, r *http.Request) {
				f(w, r.WithContext(context.WithValue(r.Context(), roomKey, p)))
			}
		}

		return func(w http.ResponseWriter, r *http.Request) {
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

			r = r.WithContext(context.WithValue(r.Context(), roomKey, p))
			f(w, r)
		}
	}
}

func (s *Service) upgradeHTTP(readBuf, writeBuf int) func(f http.HandlerFunc) http.HandlerFunc {
	return func(f http.HandlerFunc) http.HandlerFunc {
		u := &websocket.Upgrader{
			ReadBufferSize:  readBuf,
			WriteBufferSize: writeBuf,
			CheckOrigin:     func(r *http.Request) bool { return true },
		}

		return func(w http.ResponseWriter, r *http.Request) {
			p := r.Context().Value(roomKey).(*ws.Pool)
			if p.Size() == p.MaxConn {
				s.respond(w, r, ws.ErrMaxConn, http.StatusUnauthorized)
				return
			}

			c, err := p.NewConn(w, r, u)
			if err != nil {
				s.respond(w, r, err, http.StatusInternalServerError)
				return
			}

			r = r.WithContext(context.WithValue(r.Context(), upgradeKey, c))
			f(w, r)
		}
	}
}

func (s *Service) handleP2PComms() http.HandlerFunc {
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

func (s *Service) handleEcho() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := r.Context().Value(upgradeKey).(*ws.Conn)
		defer func() {
			// ! send error when Leaving session pool
			c.SendMessage("leave")

			s.l.Println("default leave")

			c.Close()
		}()

		if err := c.SendMessage("join"); err != nil {
			s.l.Println(err)
			return
		}

		s.l.Println("default join")

		for {
			var msg any
			if err := c.ReadJSON(&msg); err != nil {
				s.l.Println(err)
				return
			}

			if err := c.SendMessage(msg); err != nil {
				s.l.Println(err)
				return
			}
		}
	}
}

// ! to be discarded

func (s *Service) indexHTML(path string) http.HandlerFunc {
	render, err := t.Render(path)
	if err != nil {
		panic(err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		render(w, r, nil)
	}
}

func (s *Service) jamSessionHTML(path string) http.HandlerFunc {
	render, err := t.Render(path)
	if err != nil {
		panic(err)
	}

	// ! I should be rendering a 404 page if there is an error
	// ! in this layer, but for an MVC this will do
	return func(w http.ResponseWriter, r *http.Request) {
		uid, err := s.parseUUID(w, r)
		if err != nil {
			s.respond(w, r, err, http.StatusBadRequest)
			return
		}

		if _, err = s.c.Get(uid); err != nil {
			s.respond(w, r, err, http.StatusNotFound)
			return
		}

		render(w, r, nil)
	}
}
