package user

import (
	"context"
	"net/http"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/rog-golang-buddies/rmx/internal"
)

// var refreshTokenCookieName = "RMX_DIRECT_RT"
// var refreshTokenCookiePath = "/api/v1"

// 400 - catch all
// 401 - unauthorized
// 403 - Forbidden
// 409 - Conflict (details already exist)
// 412 - Invalid precondition
// 422 - Unprocessable

func (s *Service) handleRegistration() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var v SignupUser
		if err := s.decode(w, r, &v); err != nil {
			s.respond(w, r, err, http.StatusBadRequest)
			return
		}

		var u internal.User
		if err := v.decode(&u); err != nil {
			s.respond(w, r, err, http.StatusInternalServerError)
			return
		}

		if err := s.ur.SignUp(u); err != nil {
			s.respond(w, r, err, http.StatusInternalServerError)
			return
		}

		s.created(w, r, string(u.ID.ShortUUID()))
	}
}

func (s *Service) handleIdentity() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		email := r.Context().Value(emailKey).(string)

		u, err := s.ur.LookupEmail(internal.Email(email))
		if err != nil {
			s.respond(w, r, err, http.StatusInternalServerError)
			return
		}

		s.respond(w, r, u, http.StatusOK)
	}
}

func (s *Service) handleCreateSession(key jwk.Key) http.HandlerFunc {
	type loginUser struct {
		Email    internal.Email    `json:"email"`
		Password internal.Password `json:"password"`
	}

	type authTokens struct {
		IDToken     string `json:"idToken"`
		AccessToken string `json:"accessToken"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var v loginUser
		if err := s.decode(w, r, &v); err != nil {
			s.respond(w, r, err, http.StatusBadRequest)
			return
		}

		u, err := s.ur.LookupEmail(v.Email)
		if err != nil {
			s.respond(w, r, err, http.StatusNotFound)
			return
		}

		if err := u.Password.Compare(v.Password); err != nil {
			s.respond(w, r, err, http.StatusUnauthorized)
			return
		}

		its, ats, rts, err := s.signedTokens(key, string(u.Email), u.ID.String())
		if err != nil {
			s.respond(w, r, err, http.StatusInternalServerError)
			return
		}

		cookie := &http.Cookie{
			Path:     "/",
			Name:     cookieName,
			Value:    string(rts),
			HttpOnly: true,
			// Secure:   false,
			Secure:   r.TLS != nil,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   int(time.Hour * 24 * 7),
		}

		var data = authTokens{
			IDToken:     string(its),
			AccessToken: string(ats),
		}

		s.respondCookie(w, r, data, cookie)
	}
}

func (s *Service) handleDeleteSession() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := &http.Cookie{
			Path: "/",
			Name: cookieName,
			// Value:    "",
			HttpOnly: true,
			// Secure:   false,
			// Secure:   r.TLS != nil,
			// SameSite: http.SameSiteLaxMode,
			MaxAge: -1,
		}

		s.respondCookie(w, r, http.StatusText(http.StatusOK), c)
	}
}

// TODO still to develop
func (s *Service) handleRefreshSession(key jwk.Key) http.HandlerFunc {
	type response struct {
		AccessToken string `json:"accessToken"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		email := r.Context().Value(emailKey).(string)
		u, _ := s.ur.LookupEmail(internal.Email(email)) // NOTE assuming this exists if auth checks passed

		_, ats, rts, err := s.signedTokens(key, email, u.ID.String())
		if err != nil {
			s.respond(w, r, err, http.StatusInternalServerError)
			return
		}

		// NOTE I should be able to assume that this exists, else just renew
		var c *http.Cookie
		if c, err = r.Cookie(cookieName); err != nil {
			c = &http.Cookie{
				Path:     "/",
				Name:     cookieName,
				Value:    string(rts),
				HttpOnly: true,
				// Secure:   false, // set to true in production
				Secure:   r.TLS != nil,
				SameSite: http.SameSiteLaxMode,
				MaxAge:   int(time.Hour * 24 * 7),
			}
		} else {
			c.MaxAge = int(time.Hour * 24 * 7)
			c.Value = string(rts)
		}

		data := response{
			AccessToken: string(ats),
		}

		s.respondCookie(w, r, data, c)
	}
}

func (s *Service) authenticate(publicKey jwk.Key) func(f http.Handler) http.Handler {
	// auth.Authenticate(s, publicKey, cookieName, emailKey)

	return func(f http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			// token, err := jwt.ParseRequest(r, jwt.WithHeaderKey("Authorization"), jwt.WithHeaderKey(cookieName), jwt.WithKey(jwa.RS256, publicKey), jwt.WithValidate(true))
			token, err := jwt.ParseRequest(r, jwt.WithKey(jwa.RS256, publicKey))
			if err != nil {
				s.respond(w, r, err, http.StatusUnauthorized)
				return
			}

			email, ok := token.PrivateClaims()["email"].(string)
			if !ok {
				s.respond(w, r, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			// NOTE convert email from `string` type to `internal.Email` ?
			r = r.WithContext(context.WithValue(r.Context(), emailKey, email))
			f.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}
