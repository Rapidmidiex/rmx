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
)

func (s *Service) handleCreateRoom() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid, err := s.c.NewPool(4)
		if err != nil {
			s.respond(w, r, err, http.StatusInternalServerError)
			return
		}

		s.l.Println("create a room", s.c.Size())

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
					ID:        p.ID.ShortUUID(),
					Users:     fp.FMap(p.Keys(), func(uid suid.UUID) suid.SUID { return uid.ShortUUID() }),
					UserCount: p.Size(),
				}
			}),
		}

		s.respond(w, r, v, http.StatusOK)
	}
}

// Works with `chi.With`
func (s *Service) connectionPool(p *ws.Pool) func(f http.Handler) http.Handler {
	return func(f http.Handler) http.Handler {
		var fn func(w http.ResponseWriter, r *http.Request)
		if p != nil {
			fn = func(w http.ResponseWriter, r *http.Request) {
				f.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), roomKey, p)))
			}
		} else {
			fn = func(w http.ResponseWriter, r *http.Request) {
				s.l.Println(r.URL.Path)

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
				f.ServeHTTP(w, r)
			}
		}

		return http.HandlerFunc(fn)
	}
}

func (s *Service) upgradeHTTP(readBuf, writeBuf int) func(f http.Handler) http.Handler {
	return func(f http.Handler) http.Handler {
		u := &websocket.Upgrader{
			ReadBufferSize:  readBuf,
			WriteBufferSize: writeBuf,
			CheckOrigin:     func(r *http.Request) bool { return true },
		}

		fn := func(w http.ResponseWriter, r *http.Request) {
			p, _ := r.Context().Value(roomKey).(*ws.Pool)
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
			f.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}

// converting to handler for middleware, in order to use Chi's default type

func (s *Service) handleP2PComms() http.HandlerFunc {
	// FIXME we will change this as I know this hasn't been
	// was just my way of getting things working, not yet
	// full agreement with this.
	type response[T any] struct {
		Typ     rmx.MsgTyp `json:"type"`
		Payload T          `json:"payload"`
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
			// FIXME send error when Leaving session pool
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

		// TODO could the API be adjusted such that
		// this for-loop only needs to read and
		// never touch the code for writing
		for {
			var msg response[json.RawMessage]
			if err := c.ReadJSON(&msg); err != nil {
				s.l.Println(err)
				return
			}

			// TODO here the message will be passed off to a different handler
			// via a go routine*
			if err := c.SendMessage(response[int]{Typ: rmx.Message, Payload: 10}); err != nil {
				s.l.Println(err)
				return
			}
		}
	}
}
