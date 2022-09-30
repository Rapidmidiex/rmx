package user

import (
	"net/http"

	"github.com/rog-golang-buddies/rmx/internal"
	"github.com/rog-golang-buddies/rmx/internal/suid"
)

func (s *Service) handleUserSignUp() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var v User
		if err := s.decode(w, r, &v); err != nil {
			s.respond(w, r, err, http.StatusBadRequest)
			return
		}

		var u internal.User
		if err := v.Decode(&u); err != nil {
			s.respond(w, r, err, http.StatusInternalServerError)
			return
		}

		if err := s.r.SignUp(u); err != nil {
			s.respond(w, r, err, http.StatusInternalServerError)
			return
		}

		s.created(w, r, u.ID.String())
	}
}

func (s *Service) handleUserLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.respond(w, r, nil, http.StatusNotImplemented)
	}
}

type User struct {
	Email    internal.Email    `json:"email"`
	Username string            `json:"username"`
	Password internal.Password `json:"password"`
}

func (u User) Decode(iu *internal.User) error {
	h, err := u.Password.Hash()
	if err != nil {
		return err
	}

	*iu = internal.User{
		ID:       suid.NewUUID(),
		Email:    u.Email,
		Username: u.Username,
		Password: h,
	}

	return nil
}
