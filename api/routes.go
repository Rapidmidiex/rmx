package api

import (
	"encoding/json"
	"net/http"

	t "github.com/hyphengolang/prelude/template"

	ws "github.com/rog-golang-buddies/rapidmidiex/api/websocket"
	rmx "github.com/rog-golang-buddies/rapidmidiex/internal"
	"github.com/rog-golang-buddies/rapidmidiex/internal/suid"
)

<<<<<<< HEAD:www/routes.go
func (s *Service) handleP2PComms() http.HandlerFunc {
	type response[T any] struct {
		MessageTyp rmx.MessageType `json:"type"`
		Data       T               `json:"data"`
=======
type (
	session struct {
		Name      string      `json:"name,omitempty"`
		ID        suid.SUID   `json:"id,omitempty"`
		Users     []suid.SUID `json:"users,omitempty"`
		UserCount int         `json:"userCount"`
>>>>>>> 0f864c5a2eca7b383b3222a841413f7f644d3541:api/routes.go
	}

<<<<<<< HEAD:www/routes.go
	type join struct {
		ID        suid.SUID `json:"id"`
		SessionID suid.SUID `json:"sessionId"`
	}

	type leave struct {
		ID        suid.SUID `json:"id"`
		SessionID suid.SUID `json:"sessionId"`
		Error     any       `json:"err"`
=======
func (s *Service) routes() {
	// middleware
	s.r.Use(middleware.Logger)

	// v0
	s.r.Handle("/assets/*", s.fileServer("/assets/", "assets"))
	s.r.Get("/", s.indexHTML("ui/www/index.html"))
	s.r.Get("/play/{id}", s.jamSessionHTML("ui/www/play.html"))

	// REST v1
	s.r.Get("/api/v1/jam", s.listSessions())
	s.r.Post("/api/v1/jam", s.createSession())
	s.r.Get("/api/v1/jam/{id}", s.getSessionData)

	// Websocket
	s.r.Get("/ws/{id}", chain(s.handleJamSession(), s.upgradeHTTP, s.sessionPool))
}

func (s *Service) handleJamSession() http.HandlerFunc {
	type response struct {
		MessageType rmx.MessageType `json:"type"`
		ID          suid.SUID       `json:"id"`
		SessionID   suid.SUID       `json:"sessionId"`
>>>>>>> 0f864c5a2eca7b383b3222a841413f7f644d3541:api/routes.go
	}

	return func(w http.ResponseWriter, r *http.Request) {
		c := r.Context().Value(upgradeKey).(*ws.Conn)
		defer func() {
<<<<<<< HEAD:www/routes.go
			// ! send error when Leaving session pool
			c.SendMessage(response[leave]{
				MessageTyp: rmx.Leave,
				Data:       leave{ID: c.ID.ShortUUID(), SessionID: c.Pool().ID.ShortUUID()},
=======
			c.SendMessage(response{
				MessageType: rmx.Leave,
				ID:          c.ID.ShortUUID(),
				SessionID:   c.Pool().ID.ShortUUID(),
>>>>>>> 0f864c5a2eca7b383b3222a841413f7f644d3541:api/routes.go
			})

			c.Close()
		}()

<<<<<<< HEAD:www/routes.go
		if err := c.SendMessage(response[join]{
			MessageTyp: rmx.Join,
			Data:       join{ID: c.ID.ShortUUID(), SessionID: c.Pool().ID.ShortUUID()},
		}); err != nil {
=======
		err := c.SendMessage(response{
			MessageType: rmx.Join,
			ID:          c.ID.ShortUUID(),
			SessionID:   c.Pool().ID.ShortUUID(),
		})

		if err != nil {
>>>>>>> 0f864c5a2eca7b383b3222a841413f7f644d3541:api/routes.go
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
			if err := c.SendMessage(response[int]{MessageTyp: rmx.Message, Data: 10}); err != nil {
				s.l.Println(err)
				return
			}
		}
	}
}

func (s *Service) handleCreateRoom() http.HandlerFunc {
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

<<<<<<< HEAD:www/routes.go
func (s *Service) handleGetRoom() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid, err := s.parseUUID(w, r)
		if err != nil {
			s.respond(w, r, err, http.StatusBadRequest)
			return
		}
=======
func (s *Service) getSessionData(w http.ResponseWriter, r *http.Request) {
	uid, err := s.parseUUID(w, r)
	if err != nil {
		s.respond(w, r, err, http.StatusBadRequest)
		return
	}
>>>>>>> 0f864c5a2eca7b383b3222a841413f7f644d3541:api/routes.go

		// ! rename method as `Get` is nondescriptive
		p, err := s.c.Get(uid)
		if err != nil {
			s.respond(w, r, err, http.StatusNotFound)
			return
		}

<<<<<<< HEAD:www/routes.go
		v := &Session{
			ID:    p.ID.ShortUUID(),
			Users: rmx.FMap(p.Keys(), func(uid suid.UUID) suid.SUID { return uid.ShortUUID() }),
		}
=======
	v := &session{
		ID:    p.ID.ShortUUID(),
		Users: rmx.FMap(p.Keys(), func(uid suid.UUID) suid.SUID { return uid.ShortUUID() }),
	}
>>>>>>> 0f864c5a2eca7b383b3222a841413f7f644d3541:api/routes.go

		s.respond(w, r, v, http.StatusOK)
	}
}

func (s *Service) handleListRooms() http.HandlerFunc {
	type response struct {
		Sessions []Session `json:"sessions"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
<<<<<<< HEAD:www/routes.go
=======
		pl := s.c.List()

		sl := make([]session, 0, len(pl))
		for _, p := range pl {
			sl = append(sl, session{
				Name:      "", // name not implemented yet
				ID:        p.ID.ShortUUID(),
				UserCount: p.Size(),
			})
		}

>>>>>>> 0f864c5a2eca7b383b3222a841413f7f644d3541:api/routes.go
		v := &response{
			Sessions: rmx.FMap(s.c.List(), func(p *ws.Pool) Session { return Session{ID: p.ID.ShortUUID()} }),
		}

		s.respond(w, r, v, http.StatusOK)
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
