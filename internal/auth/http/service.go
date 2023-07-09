package http

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log"
	"net/http"
	"net/url"

	"github.com/coreos/go-oidc/v3/oidc"
	service "github.com/rapidmidiex/rmx/internal/http"
	"github.com/rapidmidiex/rmx/internal/sessions"
	"golang.org/x/oauth2"
)

type Service struct {
	ctx               context.Context
	mux               service.Service
	config            *oauth2.Config
	provider          *Provider
	loginRedirectURL  string
	logoutRedirectURL string
	logoutCallbackURL string
}

type Provider struct {
	oidc      *oidc.Provider
	domainURL string
	logoutURL string
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
	s.mux.Handle("/login/callback", s.handleLoginCallback())
	s.mux.Handle("/user", s.handleUser())
	s.mux.Handle("/logout", s.handleLogout())
	s.mux.Handle("/logout/callback", s.handleLogoutCallback())
}

func (s *Service) handleLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, err := sessions.Default(r)
		if err != nil {
			s.mux.Respond(w, r, err.Error(), http.StatusBadRequest)
			return
		}

		state, err := generateRandomState()
		if err != nil {
			s.mux.Respond(w, r, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := sess.Set(w, &sessions.Session{
			State: state,
		}); err != nil {
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

func (s *Service) handleLoginCallback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, err := sessions.Default(r)
		if err != nil {
			s.mux.Respond(w, r, err.Error(), http.StatusBadRequest)
			return
		}

		session, err := sess.Get(r)
		if err != nil {
			s.mux.Respond(w, r, err.Error(), http.StatusBadRequest)
			return
		}

		if r.URL.Query().Get("state") != session.State {
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

		if err := idToken.Claims(&session.Profile); err != nil {
			s.mux.Respond(w, r, err.Error(), http.StatusInternalServerError)
			return
		}
		session.AccessToken = token.AccessToken

		if err := sess.Set(w, session); err != nil {
			s.mux.Respond(w, r, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, s.loginRedirectURL, http.StatusTemporaryRedirect)
	}
}

func (s *Service) verifyIDToken(ctx context.Context, token *oauth2.Token) (*oidc.IDToken, error) {
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, errors.New("no id_token field in oauth2 token")
	}

	return s.provider.oidc.Verifier(&oidc.Config{ClientID: s.config.ClientID}).Verify(ctx, rawIDToken)
}

func (s *Service) handleUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, err := sessions.Default(r)
		if err != nil {
			s.mux.Respond(w, r, err.Error(), http.StatusBadRequest)
			return
		}

		session, err := sess.Get(r)
		if err != nil {
			s.mux.Respond(w, r, err.Error(), http.StatusBadRequest)
			return
		}

		s.mux.Respond(w, r, session.Profile, http.StatusOK)
	}
}

func (s *Service) handleLogout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logoutURL, err := url.Parse(s.provider.logoutURL)
		if err != nil {
			s.mux.Respond(w, r, err.Error(), http.StatusInternalServerError)
			return
		}

		parameters := url.Values{}
		parameters.Add("returnTo", s.logoutCallbackURL)
		parameters.Add("client_id", s.config.ClientID)
		logoutURL.RawQuery = parameters.Encode()

		http.Redirect(w, r, logoutURL.String(), http.StatusTemporaryRedirect)
	}
}

func (s *Service) handleLogoutCallback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, err := sessions.Default(r)
		if err != nil {
			s.mux.Respond(w, r, err.Error(), http.StatusBadRequest)
			return
		}
		// remove session
		sess.Remove(w)

		http.Redirect(w, r, s.logoutRedirectURL, http.StatusTemporaryRedirect)
	}
}

func WithContext(ctx context.Context) Option {
	return func(s *Service) {
		s.ctx = ctx
	}
}

func WithProvider(domain, clientID, clientSecret, loginCallbackURL, logoutCallbackURL string) Option {
	return func(s *Service) {
		domainURL, err := url.Parse("https://" + domain + "/")
		if err != nil {
			log.Fatalf("rmx: WithProvider\n%v", err)
		}

		logoutURL, err := url.Parse("https://" + domain + "/v2/logout")
		if err != nil {
			log.Fatalf("rmx: WithProvider\n%v", err)
		}

		provider, err := oidc.NewProvider(s.ctx, domainURL.String())
		if err != nil {
			log.Fatalf("rmx: WithProvider\n%v", err)
		}

		s.provider = &Provider{
			oidc:      provider,
			domainURL: domainURL.String(),
			logoutURL: logoutURL.String(),
		}

		s.config = &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  loginCallbackURL,
			Endpoint:     provider.Endpoint(),
			Scopes:       []string{oidc.ScopeOpenID, "profile"},
		}
		s.logoutCallbackURL = logoutCallbackURL
	}
}

// TODO: don't use options for service urls
func WithServiceURLs(loginRedirect, logoutRedirect string) Option {
	return func(s *Service) {
		s.loginRedirectURL = loginRedirect
		s.logoutRedirectURL = logoutRedirect
	}
}
