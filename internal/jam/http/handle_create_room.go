package service

import (
	"fmt"
	"net/http"
)

func (s *Service) handleCreateRoom() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// should parsing form data
		if err := r.ParseForm(); err != nil {
			// NOTE -- return form with errors
			http.Error(w, "error parsing form", http.StatusBadRequest)
			return
		}

		// var f = r.FormValue()
		// bpm=1&bpm=120&roomName=fun
		fmt.Println(r.PostForm.Encode())
		

		fmt.Fprintf(w, "ok")
	}
}
