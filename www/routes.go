package www

import (
	"net/http"

	"github.com/go-chi/chi/v5/middleware"

	t "github.com/hyphengolang/prelude/template"

	rmx "github.com/rog-golang-buddies/rapidmidiex/internal"
	"github.com/rog-golang-buddies/rapidmidiex/internal/suid"
	ws "github.com/rog-golang-buddies/rapidmidiex/www/websocket"
)

func (s Service) routes() {
	// middleware
	s.r.Use(middleware.Logger)

	// temporary static files
	// s.r.Handle("/assets/*", s.fileServer("/assets/", "assets"))
	// s.r.Get("/", s.indexHTML("ui/www/index.html"))
	// s.r.Get("/play/{id}", s.jamSessionHTML("ui/www/play.html"))

	// v0
	// s.r.Get("/api/jam/create", s.createSession())
	// s.r.Get("/api/jam/{id}", s.getSessionData())
	// s.r.HandleFunc("/jam/{id}", chain(s.handleJamSession(), s.upgradeHTTP, s.sessionPool))

	// v1
	s.r.Get("/api/v1/jam/create", s.createSession())
	s.r.Get("/api/v1/jam/{id}", s.getSessionData())
	s.r.HandleFunc("/api/v1/jam/{id}/ws", chain(s.handleJamSession(), s.upgradeHTTP, s.sessionPool))
}

func (s Service) handleJamSession() http.HandlerFunc {
	type response struct {
		MessageTyp rmx.MessageType `json:"type"`
		ID         suid.SUID       `json:"id"`
		SessionID  suid.SUID       `json:"sessionId"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		c := r.Context().Value(upgradeKey).(*ws.Conn)
		defer func() {
			c.SendMessage(response{
				MessageTyp: rmx.Leave,
				ID:         c.ID.ShortUUID(),
				SessionID:  c.Pool().ID.ShortUUID(),
			})

			c.Close()
		}()

		err := c.SendMessage(response{
			MessageTyp: rmx.Join,
			ID:         c.ID.ShortUUID(),
			SessionID:  c.Pool().ID.ShortUUID(),
		})

		if err != nil {
			s.l.Println(err)
			return
		}

		// ? could the API be adjusted such that
		// ? this for-loop only needs to read and
		// ? never touch the code for writing
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
		SessionID suid.SUID `json:"sessionId"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		uid, err := s.c.NewPool()
		if err != nil {
			s.respond(w, r, err, http.StatusInternalServerError)
			return
		}

		v := response{
			SessionID: suid.FromUUID(uid),
		}

		s.respond(w, r, v, http.StatusOK)
	}
}

func (s Service) getSessionData() http.HandlerFunc {
	type response struct {
		SessionID suid.SUID   `json:"sessionId"`
		Users     []suid.SUID `json:"users"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		uid, err := s.parseUUID(w, r, "id")
		if err != nil {
			s.respond(w, r, err, http.StatusBadRequest)
			return
		}

		// ! rename method as `Get` is undescriptive
		p, err := s.c.Get(uid)
		if err != nil {
			s.respond(w, r, err, http.StatusNotFound)
			return
		}

		v := &response{
			SessionID: p.ID.ShortUUID(),
			Users:     rmx.FMap(p.Keys(), func(uid suid.UUID) suid.SUID { return uid.ShortUUID() }),
		}

		s.respond(w, r, v, http.StatusOK)
	}
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
