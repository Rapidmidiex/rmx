package www

import (
	"net/http"

	"github.com/go-chi/chi/v5/middleware"

	t "github.com/hyphengolang/prelude/template"

	rmx "github.com/rog-golang-buddies/rapidmidiex/internal"
	"github.com/rog-golang-buddies/rapidmidiex/www/ws"
)

func (s Service) routes() {
	s.r.Use(middleware.Logger)

	// http
	s.r.Handle("/assets/*", s.fileServer("/assets/", "assets"))
	s.r.Handle("/", s.handleIndexHTML("pages/index.html"))

	// api

	// ws
	s.r.HandleFunc("/jam", chain(s.handleJamSession(), s.upgradeHTTP))
}

func (s Service) handleIndexHTML(path string) http.HandlerFunc {
	render, err := t.Render(path)
	if err != nil {
		panic(err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		render(w, r, nil)
	}
}

func (s Service) handleJamSession() http.HandlerFunc {
	type join struct {
		MessageTyp rmx.MessageTyp `json:"type"`
		ID         rmx.ID         `json:"id"`
		SessionID  rmx.ID         `json:"session_id"`
		// Users      []rmx.ID       `json:"users"`
	}

	type leave struct {
		MessageTyp rmx.MessageTyp `json:"type"`
		ID         rmx.ID         `json:"id"`
		SessionID  rmx.ID         `json:"session_id"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		c := r.Context().Value(upgradeKey).(*ws.Conn)
		defer func() {
			c.SendMessage(leave{
				MessageTyp: rmx.Leave,
				ID:         s.SafeUUID(c.ID),
				SessionID:  s.SafeUUID(c.Pool().ID),
			})

			c.Close()
		}()

		err := c.SendMessage(join{
			MessageTyp: rmx.Join,
			ID:         s.SafeUUID(c.ID),
			SessionID:  s.SafeUUID(c.Pool().ID),
			// grabbing user info should be handled by a proper RESTful endpoint
			// Can be used alongside JS fetch API
			// Users: n/a,
		})

		if err != nil {
			s.l.Println(err)
			return
		}

		for {
			var n int
			if err := c.ReadJSON(&n); err != nil {
				s.l.Println(err)
				return
			}

			if err := c.SendMessage(n + 10); err != nil {
				s.l.Println(err)
				return
			}
		}
	}
}
