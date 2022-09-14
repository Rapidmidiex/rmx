package www

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

func (s Service) upgradeHTTP(f http.HandlerFunc) http.HandlerFunc {
	u := &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if s.p.Size() == s.p.MaxConn {
			err := fmt.Errorf("pool: maximum number of connections reached")
			s.respond(w, r, err, http.StatusUnauthorized)
			return
		}

		c, err := s.p.NewConn(w, r, u)
		if err != nil {
			s.respond(w, r, err, http.StatusInternalServerError)
			return
		}

		r = r.WithContext(context.WithValue(r.Context(), upgradeKey, c))
		f(w, r)
	}
}
