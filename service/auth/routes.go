package auth

import (
	"context"
	"net/http"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/rog-golang-buddies/rmx/internal/dto"
	"github.com/rog-golang-buddies/rmx/service/internal/auth"
)

// Account Sign Up request handler
func (s *Service) handleSignUp() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var u dto.User
		if err := s.decode(w, r, &u); err != nil {
			s.respond(w, r, err, http.StatusBadRequest)
			return
		}

		// check if the provided string is a valid email
		if err := u.Email.Validate(); err != nil {
			s.respond(w, r, err, http.StatusBadRequest)
			return
		}

		// check if the password is strong enough
		if err := u.Password.Validate(); err != nil {
			s.respond(w, r, err, http.StatusBadRequest)
			return
		}

		// hash the password to store in db
		if err := u.HashPassword(); err != nil {
			s.respond(w, r, err, http.StatusInternalServerError)
			return
		}

		if err := s.ur.Add(&u); err != nil {
			s.respond(w, r, err, http.StatusInternalServerError)
			return
		}

		s.created(w, r, string(u.ID.ShortUUID()))
	}
}

// Account Sign In handler
func (s *Service) handleSignIn(key jwk.Key) http.HandlerFunc {
	type loginUser struct {
		Email    dto.Email    `json:"email"`
		Password dto.Password `json:"password"`
	}

	type authTokens struct {
		IDToken     string `json:"idToken"`
		AccessToken string `json:"accessToken"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var u loginUser
		if err := s.decode(w, r, &u); err != nil {
			s.respond(w, r, err, http.StatusBadRequest)
			return
		}

		// Get users' information
		ui, err := s.ur.LookupEmail(string(u.Email))
		if err != nil {
			s.respond(w, r, err, http.StatusNotFound)
			return
		}

		// Check if the provided password matches the users'
		if err := ui.ComparePassword(string(u.Password)); err != nil {
			s.respond(w, r, err, http.StatusUnauthorized)
			return
		}

		// Generate new JWT tokens
		its, ats, rts, err := s.signedTokens(key, string(ui.Email), ui.ID.String())
		if err != nil {
			s.respond(w, r, err, http.StatusInternalServerError)
			return
		}

		// NOTE: Secure should be set to true in production
		cookie := &http.Cookie{
			Path:     "/",
			Name:     auth.RefreshTokenCookieName,
			Value:    string(rts),
			HttpOnly: true,
			Secure:   r.TLS != nil,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   int(auth.RefreshTokenExpiry),
		}

		v := &authTokens{
			IDToken:     string(its),
			AccessToken: string(ats),
		}

		s.respondCookie(w, r, v, cookie)
	}
}

// Account Sign Out handler
// removes the Refresh Token by setting its MaxAge property to -1
func (s *Service) handleSignOut() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// NOTE: Secure should be set to true in production
		c := &http.Cookie{
			Path:     "/",
			Name:     auth.RefreshTokenCookieName,
			Value:    "",
			HttpOnly: true,
			Secure:   r.TLS != nil,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   -1,
		}

		s.respondCookie(w, r, http.StatusText(http.StatusOK), c)
	}
}

func (s *Service) handleRefreshToken(key jwk.Key) http.HandlerFunc {
	type response struct {
		AccessToken string `json:"accessToken"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		rtc, err := r.Cookie(auth.RefreshTokenCookieName)
		if err != nil {
			s.respond(w, r, err, http.StatusUnauthorized)
			return
		}

		if err := s.arc.ValidateRefreshToken(context.Background(), rtc.Value); err != nil {
			s.respond(w, r, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		}

		tc, err := auth.ParseRefreshTokenWithValidate(&key, rtc.Value)
		if err != nil {
			s.respond(w, r, err, http.StatusUnauthorized)
			return
		}

		email, ok := tc.PrivateClaims()["email"].(string)
		if !ok {
			s.respond(w, r, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		ui, err := s.ur.LookupEmail(email)
		if err != nil {
			s.respond(w, r, err, http.StatusInternalServerError)
			return
		}

		_, ats, rts, err := s.signedTokens(key, email, ui.ID.String())
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

		v := &response{
			AccessToken: string(ats),
		}

		s.respondCookie(w, r, v, c)
	}
}

func (s *Service) handleUserInfo() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		email := r.Context().Value(auth.EmailKey).(string)

		u, err := s.ur.LookupEmail(email)
		if err != nil {
			s.respond(w, r, err, http.StatusInternalServerError)
			return
		}

		s.respond(w, r, u, http.StatusOK)
	}
}
