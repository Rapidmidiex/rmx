package http

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	service "github.com/rapidmidiex/rmx/internal/http"
	"github.com/rapidmidiex/rmx/internal/sessions"
	"golang.org/x/oauth2"
)

type Service struct {
	ctx      context.Context
	mux      service.Service
	config   *oauth2.Config
	provider *oidc.Provider
	ss       *sessions.Store
}

type Option func(*Service)

func New(opts ...Option) *Service {
	s := Service{
		mux: service.New(),
	}

	for _, opt := range opts {
		opt(&s)
	}

	s.routes()
	return &s
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Service) routes() {
	s.mux.Handle("/login", s.handleLogin())
	s.mux.Handle("/callback", s.handleCallback())
	s.mux.Handle("/user", s.handleUser())
	s.mux.Handle("/logout", s.handleLogout())
}

func (s *Service) handleLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state, err := generateRandomState()
		if err != nil {
			s.mux.Respond(w, r, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := s.ss.Set(w, "state", state); err != nil {
			s.mux.Respond(w, r, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, s.config.AuthCodeURL(state), http.StatusTemporaryRedirect)
	}
}

func generateRandomState() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	state := base64.StdEncoding.EncodeToString(b)

	return state, nil
}

func (s *Service) handleCallback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessState, err := s.ss.Get(r, "state")
		if err != nil {
			s.mux.Respond(w, r, err.Error(), http.StatusBadRequest)
			return
		}

		if r.URL.Query().Get("state") != sessState {
			s.mux.Respond(w, r, nil, http.StatusBadRequest)
			return
		}

		// Exchange an authorization code for a token.
		token, err := s.config.Exchange(r.Context(), r.URL.Query().Get("code"))
		if err != nil {
			s.mux.Respond(w, r, err.Error(), http.StatusUnauthorized)
			return
		}

		idToken, err := s.verifyIDToken(r.Context(), token)
		if err != nil {
			s.mux.Respond(w, r, err.Error(), http.StatusInternalServerError)
			return
		}

		var profile map[string]any
		if err := idToken.Claims(&profile); err != nil {
			s.mux.Respond(w, r, err.Error(), http.StatusInternalServerError)
			return
		}

		s.mux.Respond(w, r, profile, http.StatusOK)
	}
}

func (s *Service) verifyIDToken(ctx context.Context, token *oauth2.Token) (*oidc.IDToken, error) {
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, errors.New("no id_token field in oauth2 token")
	}

	return s.provider.Verifier(&oidc.Config{ClientID: s.config.ClientID}).Verify(ctx, rawIDToken)
}

func (s *Service) handleUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

func (s *Service) handleLogout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

func WithContext(ctx context.Context) Option {
	return func(s *Service) {
		s.ctx = ctx
	}
}

func WithProvider(domain, clientID, clientSecret, callbackURL string) Option {
	return func(s *Service) {
		provider, err := oidc.NewProvider(
			s.ctx,
			fmt.Sprint("https://", domain, "/"),
		)
		if err != nil {
			log.Fatalf("rmx: WithProvider\n%v", err)
		}

		s.provider = provider

		s.config = &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  callbackURL,
			Endpoint:     provider.Endpoint(),
			Scopes:       []string{oidc.ScopeOpenID, "profile"},
		}
	}
}

func WithSessionStore(ss *sessions.Store) Option {
	return func(s *Service) {
		s.ss = ss
	}
}
