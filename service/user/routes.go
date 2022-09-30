package user

import "net/http"

func (s *Service) handleUserSignUp() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.respond(w, r, nil, http.StatusNotImplemented)
	}
}

func (s *Service) handleUserLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.respond(w, r, nil, http.StatusNotImplemented)
	}
}
