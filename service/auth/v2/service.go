package auth

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"

	h "github.com/hyphengolang/prelude/http"

	"github.com/rog-golang-buddies/rmx/internal"
	"github.com/rog-golang-buddies/rmx/internal/auth"
	"github.com/rog-golang-buddies/rmx/internal/fp"
	"github.com/rog-golang-buddies/rmx/internal/suid"
	// big no-no
)

var (
	ErrNoCookie        = errors.New("user: cookie not found")
	ErrSessionNotFound = errors.New("user: session not found")
	ErrSessionExists   = errors.New("user: session already exists")
)

/*
Register a new user

	[?] POST /auth/sign-up

Get current account identity

	[?] GET /account/me

Create a cookie

	[?] POST /auth/sign-in

Delete a cookie

	[?] DELETE /auth/sign-out

Refresh token

	[ ] GET /auth/refresh
*/
func (s *Service) routes() {
	key := auth.NewPairES256()

	s.m.Route("/api/v2/auth", func(r chi.Router) {
		r.Post("/sign-in", s.handleSignIn(key.Private()))
		r.Delete("/sign-out", s.handleSignOut())
		r.Post("/sign-up", s.handleSignUp())

		r.Get("/refresh", s.handleRefresh(key.Private()))
	})

	s.m.Route("/api/v2/account", func(r chi.Router) {
		auth := r.With(auth.Authenticate(jwa.ES256, key.Public()))
		auth.Get("/me", s.handleIdentity())
	})
}

func (s *Service) handleRefresh(key jwk.Key) http.HandlerFunc {
	type token struct {
		AccessToken string `json:"accessToken"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(auth.RefreshTokenCookieName)
		if err != nil {
			s.respond(w, r, err, http.StatusUnauthorized)
			return
		}

		s.setCookie(w, c)
		s.respond(w, r, nil, http.StatusNotImplemented)
	}
}

func (s *Service) handleIdentity() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		email := r.Context().Value(internal.EmailKey).(internal.Email)

		u, err := s.r.Select(r.Context(), email)
		if err != nil {
			s.respond(w, r, err, http.StatusNotFound)
			return
		}

		s.respond(w, r, u, http.StatusOK)
	}
}

func (s *Service) handleSignIn(privateKey jwk.Key) http.HandlerFunc {
	type token struct {
		IDToken     string `json:"idToken"`
		AccessToken string `json:"accessToken"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var dto User
		if err := s.decode(w, r, &dto); err != nil {
			s.respond(w, r, err, http.StatusBadRequest)
			return
		}

		u, err := s.r.Select(r.Context(), dto.Email)
		if err != nil {
			s.respond(w, r, err, http.StatusNotFound)
			return
		}

		if err := u.Password.Compare(dto.Password.String()); err != nil {
			s.respond(w, r, err, http.StatusUnauthorized)
			return
		}

		its, ats, rts, err := s.signedTokens(privateKey, u.Email.String(), suid.NewUUID())
		if err != nil {
			s.respond(w, r, err, http.StatusInternalServerError)
			return
		}

		c := &http.Cookie{
			Path:     "/",
			Name:     auth.RefreshTokenCookieName,
			Value:    string(rts),
			HttpOnly: true,
			Secure:   r.TLS != nil,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   int(auth.RefreshTokenExpiry),
		}

		tk := &token{
			IDToken:     string(its),
			AccessToken: string(ats),
		}

		s.setCookie(w, c)
		s.respond(w, r, tk, http.StatusOK)
	}
}

func (s *Service) handleSignOut() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := &http.Cookie{
			Path:     "/",
			Name:     auth.RefreshTokenCookieName,
			HttpOnly: true,
			// Secure:   r.TLS != nil,
			// SameSite: http.SameSiteLaxMode,
			MaxAge: -1,
		}

		s.setCookie(w, c)
		s.respond(w, r, http.StatusText(http.StatusOK), http.StatusOK)
	}
}

func (s *Service) handleSignUp() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var u internal.User
		if err := s.newUser(w, r, &u); err != nil {
			s.respond(w, r, err, http.StatusBadRequest)
			return
		}

		if err := s.r.Insert(r.Context(), &u); err != nil {
			s.respond(w, r, err, http.StatusInternalServerError)
			return
		}

		s.created(w, r, u.ID.ShortUUID().String())
	}
}

type Service struct {
	ctx context.Context

	m chi.Router
	r internal.UserRepo

	log  func(...any)
	logf func(string, ...any)

	decode    func(http.ResponseWriter, *http.Request, any) error
	respond   func(http.ResponseWriter, *http.Request, any, int)
	created   func(http.ResponseWriter, *http.Request, string)
	setCookie func(http.ResponseWriter, *http.Cookie)
}

func (s *Service) newUser(w http.ResponseWriter, r *http.Request, u *internal.User) (err error) {
	var dto User
	if err = s.decode(w, r, &dto); err != nil {
		return
	}

	var h internal.PasswordHash
	h, err = dto.Password.Hash()
	if err != nil {
		return
	}

	*u = internal.User{
		ID:       suid.NewUUID(),
		Username: dto.Username,
		Email:    dto.Email,
		Password: h,
	}

	return nil
}

func (s *Service) parseUUID(w http.ResponseWriter, r *http.Request) (suid.UUID, error) {
	return suid.ParseString(chi.URLParam(r, "uuid"))
}

// TODO there is two cid's being used here, need clarification
func (s *Service) signedTokens(key jwk.Key, email string, uuid suid.UUID) (its, ats, rts []byte, err error) {
	opt := auth.TokenOption{
		Issuer:     "github.com/rog-golang-buddies/rmx",
		Subject:    uuid.String(), // new client ID for tracking user connections
		Expiration: time.Hour * 10,
		Claims:     []fp.Tuple{{"email", email}},
		Algo:       jwa.ES256,
	}

	if its, err = auth.SignToken(key, &opt); err != nil {
		return nil, nil, nil, err
	}

	opt.Expiration = time.Minute * 5
	if ats, err = auth.SignToken(key, &opt); err != nil {
		return nil, nil, nil, err
	}

	opt.Expiration = time.Hour * 24 * 7
	if rts, err = auth.SignToken(key, &opt); err != nil {
		return nil, nil, nil, err
	}

	return its, ats, rts, nil
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.m.ServeHTTP(w, r) }

func NewService(ctx context.Context, m chi.Router, r internal.UserRepo) *Service {
	s := &Service{
		ctx,

		m,
		r,

		log.Println,
		log.Printf,

		h.Decode,
		h.Respond,
		h.Created,
		http.SetCookie,
	}

	s.routes()
	return s
}

func (s *Service) Context() context.Context {
	if s.ctx != nil {
		return s.ctx
	}
	return context.Background()
}

type User struct {
	Email    internal.Email    `json:"email"`
	Username string            `json:"username"`
	Password internal.Password `json:"password"`
}
