package api

import "net/http"
import t "github.com/hyphengolang/prelude/template"
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
