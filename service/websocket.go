package service

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/websocket"

	rmx "github.com/rog-golang-buddies/rapidmidiex/internal"
	"github.com/rog-golang-buddies/rapidmidiex/internal/suid"
	ws "github.com/rog-golang-buddies/rapidmidiex/internal/websocket"
)

func (s Service) connectionPool(p *ws.Pool) func(f http.HandlerFunc) http.HandlerFunc {
	// * testing/debugging
	if p != nil {
		return func(f http.HandlerFunc) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				s.l.Println("default for ping")
				r = r.WithContext(context.WithValue(r.Context(), roomKey, p))
				f(w, r)
			}
		}
	}

	return func(f http.HandlerFunc) http.HandlerFunc {
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

func (s Service) upgradeHTTP(readBuf, writeBuf int) func(f http.HandlerFunc) http.HandlerFunc {
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

func (s Service) handleEcho() http.HandlerFunc {
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
