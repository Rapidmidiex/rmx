package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/lestrrat-go/jwx/v2/jwk"

	"github.com/hyphengolang/prelude/types/email"
	"github.com/hyphengolang/prelude/types/password"

	// github.com/rog-golang-buddies/rmx/service/internal/auth/auth
	"github.com/hyphengolang/prelude/types/suid"
	"github.com/rog-golang-buddies/rmx/internal"
	"github.com/rog-golang-buddies/rmx/store/user"

	"github.com/rog-golang-buddies/rmx/pkg/auth"
	"github.com/rog-golang-buddies/rmx/pkg/service"
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

Delete devices linked to account

	[ ] DELETE /account/{uuid}/device

this returns a list of current connections:

	[ ] GET /account/{uuid}/devices

Create a cookie

	[?] POST /auth/sign-in

Delete a cookie

	[?] DELETE /auth/sign-out

Refresh token

	[?] GET /auth/refresh
*/
type Service struct {
	service.Service

	r  user.Repo
	tc internal.TokenClient
}

func (s *Service) routes() {
	public, private := auth.ES256()

	s.Route("/api/v1/auth", func(r chi.Router) {
		r.Post("/sign-in", s.handleSignIn(private))
		r.Delete("/sign-out", s.handleSignOut())
		r.Post("/sign-up", s.handleSignUp())

		r.Get("/refresh", s.handleRefresh(public, private))
	})

	s.Route("/api/v1/account", func(r chi.Router) {
		r.Get("/me", s.handleIdentity(public))
	})
}

// FIXME this endpoint is broken due to the redis client
// We need to try fix this ASAP
func (s *Service) handleRefresh(public, private jwk.Key) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// NOTE temp switch away from auth middleware
		jtk, err := auth.ParseCookie(r, public, cookieName)
		if err != nil {
			s.Respond(w, r, err, http.StatusUnauthorized)
			return
		}

		claim, ok := jtk.PrivateClaims()["email"].(string)
		if !ok {
			s.RespondText(w, r, http.StatusInternalServerError)
			return
		}

		u, err := s.r.Select(r.Context(), email.Email(claim))
		if err != nil {
			s.Respond(w, r, err, http.StatusForbidden)
			return
		}

		// FIXME commented out as not complete
		// // already checked in auth but I am too tired
		// // to come up with a cleaner solution
		// k, _ := r.Cookie(cookieName)

		// err := s.tc.ValidateRefreshToken(r.Context(), k.Value)
		// if err != nil {
		// 	s.Respond(w, r, err, http.StatusInternalServerError)
		// 	return
		// }

		// // token validated, now it should be set inside blacklist
		// // this prevents token reuse
		// err = s.tc.BlackListRefreshToken(r.Context(), k.Value)
		// if err != nil {
		// 	s.Respond(w, r, err, http.StatusInternalServerError)
		// }

		// cid := j.Subject()
		// _, ats, rts, err := s.signedTokens(private, claim.String(), suid.SUID(cid))
		// if err != nil {
		// 	s.Respond(w, r, err, http.StatusInternalServerError)
		// 	return
		// }

		u.ID, _ = suid.ParseString(jtk.Subject())

		_, ats, rts, err := s.signedTokens(private, u)
		if err != nil {
			s.Respond(w, r, err, http.StatusInternalServerError)
			return
		}

		c := s.newCookie(w, r, string(rts), auth.RefreshTokenExpiry)

		tk := &Token{
			AccessToken: string(ats),
		}

		s.SetCookie(w, c)
		s.Respond(w, r, tk, http.StatusOK)
	}
}

func (s *Service) handleIdentity(public jwk.Key) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, err := s.authenticate(w, r, public)
		if err != nil {
			s.Respond(w, r, err, http.StatusUnauthorized)
			return
		}

		s.Respond(w, r, u, http.StatusOK)
	}
}

func (s *Service) handleSignIn(privateKey jwk.Key) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var dto User
		if err := s.Decode(w, r, &dto); err != nil {
			s.Respond(w, r, err, http.StatusBadRequest)
			return
		}

		u, err := s.r.Select(r.Context(), dto.Email)
		if err != nil {
			s.Respond(w, r, err, http.StatusNotFound)
			return
		}

		if err := u.Password.Compare(dto.Password.String()); err != nil {
			s.Respond(w, r, err, http.StatusUnauthorized)
			return
		}

		// NOTE - need to replace u.UUID with a client based ID
		// this will mean different cookies for multi-device usage
		u.ID = suid.NewUUID()

		its, ats, rts, err := s.signedTokens(privateKey, u)
		if err != nil {
			s.Respond(w, r, err, http.StatusInternalServerError)
			return
		}

		c := s.newCookie(w, r, string(rts), auth.RefreshTokenExpiry)

		tk := &Token{
			IDToken:     string(its),
			AccessToken: string(ats),
		}

		s.SetCookie(w, c)
		s.Respond(w, r, tk, http.StatusOK)
	}
}

func (s *Service) handleSignOut() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := s.newCookie(w, r, "", -1)

		s.SetCookie(w, c)
		s.Respond(w, r, http.StatusText(http.StatusOK), http.StatusOK)
	}
}

func (s *Service) handleSignUp() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var u internal.User
		if err := s.newUser(w, r, &u); err != nil {
			s.Respond(w, r, err, http.StatusBadRequest)
			return
		}

		if err := s.r.Insert(r.Context(), &u); err != nil {
			s.Respond(w, r, err, http.StatusInternalServerError)
			return
		}

		suid := u.ID.ShortUUID().String()
		s.Created(w, r, suid)
	}
}

func (s *Service) newUser(w http.ResponseWriter, r *http.Request, u *internal.User) (err error) {
	var dto User
	if err = s.Decode(w, r, &dto); err != nil {
		return
	}

	var h password.PasswordHash
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

func (s *Service) newCookie(w http.ResponseWriter, r *http.Request, value string, maxAge time.Duration) *http.Cookie {
	c := &http.Cookie{
		Path:     "/",
		Name:     cookieName,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(maxAge),
		Value:    string(value),
	}
	return c
}

func (s *Service) authenticate(w http.ResponseWriter, r *http.Request, public jwk.Key) (*internal.User, error) {
	tk, err := auth.ParseRequest(r, public)
	if err != nil {
		return nil, err
	}

	claim, ok := tk.PrivateClaims()["email"].(string)
	if err := fmt.Errorf("email claim does not exist"); !ok {
		return nil, err
	}

	u, err := s.r.Select(r.Context(), email.MustParse(claim))
	if err != nil {
		return nil, err
	}

	return u, nil
}

// TODO there is two cid's being used here, need clarification
func (s *Service) signedTokens(private jwk.Key, u *internal.User) (its, ats, rts []byte, err error) {
	o := auth.TokenOption{
		Issuer:  issuer,
		Subject: u.ID.ShortUUID().String(), // new client ID for tracking user connections
		// Audience: []string{},
		Claims: map[string]any{"email": u.Email},
	}

	// its
	o.Expiration = idTokenExp
	if its, err = auth.Sign(private, &o); err != nil {
		return
	}

	// ats
	o.Expiration = accessTokenExp
	if ats, err = auth.Sign(private, &o); err != nil {
		return
	}

	// rts
	o.Expiration = refreshTokenExp
	if rts, err = auth.Sign(private, &o); err != nil {
		return
	}

	return
}

func NewService(ctx context.Context, m chi.Router, r user.Repo, tc internal.TokenClient) *Service {
	s := &Service{service.New(ctx, m), r, tc}
	s.routes()
	return s
}

type User struct {
	Email    email.Email       `json:"email"`
	Username string            `json:"username"`
	Password password.Password `json:"password"`
}

type Token struct {
	IDToken     string `omitempty,json:"idToken"`
	AccessToken string `omitempty,json:"accessToken"`
}

const (
	issuer          = "github.com/rog-golang-buddies/rmx"
	cookieName      = "RMX_REFRESH_TOKEN"
	idTokenExp      = time.Hour * 10
	refreshTokenExp = time.Hour * 24 * 7
	accessTokenExp  = time.Minute * 5
)
