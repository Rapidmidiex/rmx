package service

import (
	"net/http"

	"github.com/hyphengolang/prelude/types/email"
	"github.com/hyphengolang/prelude/types/password"
	"github.com/hyphengolang/prelude/types/suid"
	service "github.com/rog-golang-buddies/rmx/common/http"
	"github.com/rog-golang-buddies/rmx/common/sql"
	"github.com/rog-golang-buddies/rmx/domain/user"
)

type userService struct {
	mux service.Service
	r   sql.RWRepo[user.User]
}

func (s *userService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func NewService(r sql.RWRepo[user.User]) http.Handler {
	s := &userService{
		mux: service.New(),
		r:   r,
	}

	s.routes()
	return s
}

func (s *userService) routes() {
	s.mux.Get("/register", s.handleRegister())
	s.mux.Delete("/register", s.handleUnregister())
	s.mux.Get("/ping", s.handleHealth())
}

func (s *userService) handleRegister() http.HandlerFunc {
	type User struct {
		Email    email.Email       `json:"email"`
		Username string            `json:"username"`
		Password password.Password `json:"password"`
	}

	newUser := func(w http.ResponseWriter, r *http.Request, u *user.User) (err error) {
		var dto User
		if err = s.mux.Decode(w, r, &dto); err != nil {
			return
		}

		var h password.PasswordHash
		h, err = dto.Password.Hash()
		if err != nil {
			return
		}

		*u = user.User{
			ID:       suid.NewUUID(),
			Username: dto.Username,
			Email:    dto.Email,
			Password: h,
		}

		return nil
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var u user.User
		if err := newUser(w, r, &u); err != nil {
			s.mux.Respond(w, r, err, http.StatusBadRequest)
			return
		}

		if err := s.r.Write(r.Context(), &u); err != nil {
			s.mux.Respond(w, r, err, http.StatusInternalServerError)
			return
		}

		suid := u.ID.ShortUUID().String()
		s.mux.Created(w, r, suid)
	}
}

func (s *userService) handleUnregister() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.mux.Respond(w, r, "unregister", http.StatusOK)
	}
}

func (s *userService) handleHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.mux.Respond(w, r, "ping", http.StatusOK)
	}
}
