package www

import (
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	suid "github.com/lithammer/shortuuid/v4"

	t "github.com/hyphengolang/prelude/template"

	rmx "github.com/rog-golang-buddies/rapidmidiex/internal"
	ws "github.com/rog-golang-buddies/rapidmidiex/www/ws"
)

func (s Service) routes() {
	// middleware
	s.r.Use(middleware.Logger)

	// http
	s.r.Handle("/assets/*", s.fileServer("/assets/", "assets"))
	s.r.Get("/", s.indexHTML("pages/index.html"))
	s.r.Get("/play/{id}", s.jamSessionHTML("pages/play.html"))

	// api
	s.r.Get("/api/jam/create", s.createSession())
	s.r.Get("/api/jam/{id}", s.getSessionData())

	// ws
	s.r.HandleFunc("/jam/{id}", chain(s.handleJamSession(), s.upgradeHTTP, s.sessionPool))
}

func (s Service) indexHTML(path string) http.HandlerFunc {
	render, err := t.Render(path)
	if err != nil {
		panic(err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		render(w, r, nil)
	}
}

func (s Service) jamSessionHTML(path string) http.HandlerFunc {
	render, err := t.Render(path)
	if err != nil {
		panic(err)
	}

	// ! I should be rendering a 404 page if there is an error
	// ! in this layer, but for an MVC this will do
	return func(w http.ResponseWriter, r *http.Request) {
		uid, err := s.parseUUID(w, r, "id")
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

func (s Service) handleJamSession() http.HandlerFunc {
	type response struct {
		MessageTyp rmx.MessageTyp `json:"type"`
		ID         rmx.ID         `json:"id"`
		SessionID  rmx.ID         `json:"sessionId"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		c := r.Context().Value(upgradeKey).(*ws.Conn)
		defer func() {
			c.SendMessage(response{
				MessageTyp: rmx.Leave,
				ID:         s.safeUUID(c.ID),
				SessionID:  s.safeUUID(c.Pool().ID),
			})

			c.Close()
		}()

		err := c.SendMessage(response{
			MessageTyp: rmx.Join,
			ID:         s.safeUUID(c.ID),
			SessionID:  s.safeUUID(c.Pool().ID),
		})

		if err != nil {
			s.l.Println(err)
			return
		}

		// ?could the API be adjusted such that
		// ?this for-loop only needs to read and
		// ?never touch the code for writing
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

func (s Service) createSession() http.HandlerFunc {
	type response struct {
		SessionID rmx.ID `json:"sessionId"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		uid, err := s.c.NewPool()
		if err != nil {
			s.respond(w, r, err, http.StatusInternalServerError)
			return
		}

		v := response{
			SessionID: rmx.ID(suid.DefaultEncoder.Encode(uid)),
		}

		s.respond(w, r, v, http.StatusOK)
	}
}

func (s Service) getSessionData() http.HandlerFunc {
	type response struct {
		SessionID rmx.ID   `json:"sessionId"`
		Users     []rmx.ID `json:"users"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		uid, err := s.parseUUID(w, r, "id")
		if err != nil {
			s.respond(w, r, err, http.StatusBadRequest)
			return
		}

		// !rename method as `Get` is undescriptive
		p, err := s.c.Get(uid)
		if err != nil {
			s.respond(w, r, err, http.StatusNotFound)
			return
		}

		v := &response{
			SessionID: rmx.ID(suid.DefaultEncoder.Encode(p.ID)),
			Users: rmx.FMap(p.Keys(), func(uid uuid.UUID) rmx.ID {
				return rmx.ID(suid.DefaultEncoder.Encode(uid))
			}),
		}

		s.respond(w, r, v, http.StatusOK)
	}
}
