package www

import (
	"net/http"

	t "github.com/hyphengolang/prelude/template"
	"github.com/rog-golang-buddies/rapidmidiex/www/ws"
)

func (s Service) routes() {
	// http
	s.r.Handle("/assets/*", s.fileServer("/assets/", "assets"))
	s.r.Handle("/", s.handleIndexHTML("pages/index.html"))

	// ws
	s.r.HandleFunc("/jam", chain(s.handleJamSessionWS, s.upgradeHTTP))
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

func (s Service) handleJamSessionWS(w http.ResponseWriter, r *http.Request) {
	c := r.Context().Value(upgradeKey).(*ws.Conn)
	defer c.Close()

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
