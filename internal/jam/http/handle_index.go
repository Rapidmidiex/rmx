package service

import (
	"net/http"

	"github.com/rapidmidiex/rmx/html"
)

func (s *Service) handleIndex() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"Title": "Welcome",
		}

		if err := html.Execute(w, "index", data); err != nil {
			http.Error(w, "error writing partial "+err.Error(), http.StatusInternalServerError)
		}
	}
}