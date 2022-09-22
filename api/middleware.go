package api

import (
	"context"
	"net/http"

	"github.com/gorilla/websocket"

	ws "github.com/rog-golang-buddies/rapidmidiex/api/websocket"
)

func (s Service) connectionPool(f http.HandlerFunc) http.HandlerFunc {
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
