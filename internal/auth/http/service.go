package http

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/rapidmidiex/rmx/internal/auth"
	service "github.com/rapidmidiex/rmx/internal/http"
	"github.com/rapidmidiex/rmx/internal/sessions"
	"golang.org/x/oauth2"
)

type Service struct {
	ctx          context.Context
	mux          service.Service
	config       *oauth2.Config
	provider     *Provider
	callbacktURL string
	redirectURL  string
}

type Provider struct {
	oidc      *oidc.Provider
	domainURL string
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
	s.mux.Handle("/logout", s.handleLogout())
	s.mux.Handle("/user", auth.IsAuthenticated(s.handleUser()))
}

func (s *Service) handleLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state, err := generateRandomState()
		if err != nil {
			s.mux.Respond(w, r, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := sessions.SetSession(w, r, &sessions.Session{
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
		session, err := sessions.GetSession(r)
		if err != nil {
			s.mux.Respond(w, r, err.Error(), http.StatusBadRequest)
			return
		}

		if r.URL.Query().Get("state") != session.State {
			s.mux.Respond(w, r, "state parameter doesn't match", http.StatusForbidden)
			return
		}

		// Exchange an authorization code for a token.
		token, err := s.config.Exchange(r.Context(), r.URL.Query().Get("code"))
		if err != nil {
			s.mux.Respond(w, r, err.Error(), http.StatusUnauthorized)
			return
		}

		fmt.Println(token.AccessToken)

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
		session.State = ""

		if err := sessions.SetSession(w, r, session); err != nil {
			s.mux.Respond(w, r, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, s.redirectURL, http.StatusTemporaryRedirect)
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
		session, err := sessions.GetSession(r)
		if err != nil {
			s.mux.Respond(w, r, err.Error(), http.StatusBadRequest)
			return
		}

		s.mux.Respond(w, r, session.Profile, http.StatusOK)
	}
}

func (s *Service) handleLogout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// remove session
		if err := sessions.RemoveFromRequest(w, r); err != nil {
			s.mux.Respond(w, r, err.Error(), http.StatusBadRequest)
			return
		}

		logoutURL, err := url.Parse(s.provider.domainURL + "v2/logout")
		if err != nil {
			s.mux.Respond(w, r, err.Error(), http.StatusInternalServerError)
			return
		}

		parameters := url.Values{}
		parameters.Add("returnTo", s.redirectURL)
		parameters.Add("client_id", s.config.ClientID)
		logoutURL.RawQuery = parameters.Encode()

		http.Redirect(w, r, logoutURL.String(), http.StatusTemporaryRedirect)
	}
}

func WithContext(ctx context.Context) Option {
	return func(s *Service) {
		s.ctx = ctx
	}
}

type CustomClaims struct {
	Scope string `json:"scope"`
}

func (c CustomClaims) Validate(ctx context.Context) error {
	return nil
}

func WithProvider(domain, clientID, clientSecret, callbackURL string, audience []string) Option {
	return func(s *Service) {
		domainURL, err := url.Parse("https://" + domain + "/")
		if err != nil {
			log.Fatalf("rmx: WithProvider failed to parse domain url\n%v", err)
		}

		provider, err := oidc.NewProvider(s.ctx, domainURL.String())
		if err != nil {
			log.Fatalf("rmx: WithProvider failed to initialize new provider\n%v", err)
		}

		keysetProvider := jwks.NewCachingProvider(domainURL, 5*time.Minute)
		jwtValidator, err := validator.New(
			keysetProvider.KeyFunc,
			validator.RS256,
			domainURL.String(),
			audience,
			validator.WithCustomClaims(func() validator.CustomClaims {
				return &CustomClaims{}
			}),
			validator.WithAllowedClockSkew(time.Minute),
		)
		if err != nil {
			log.Fatalf("rmx: WithProvider failed to set up jwt validator")
		}

		auth.Validator = jwtValidator

		s.provider = &Provider{
			oidc:      provider,
			domainURL: domainURL.String(),
		}

		s.config = &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  callbackURL,
			Endpoint:     provider.Endpoint(),
			Scopes:       []string{oidc.ScopeOpenID, "profile"},
		}
		s.callbacktURL = callbackURL
	}
}

// TODO: don't use options for service urls
func WithCallbackURL(url string) Option {
	return func(s *Service) {
		s.redirectURL = url
	}
}
