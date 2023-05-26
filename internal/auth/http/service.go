package http

import (
	"context"
	"database/sql"
	"log"
	"net/http"

	"github.com/rapidmidiex/rmx/internal/auth"
	authDB "github.com/rapidmidiex/rmx/internal/auth/postgres"
	"github.com/rapidmidiex/rmx/internal/auth/provider"
	service "github.com/rapidmidiex/rmx/internal/http"
	"github.com/rapidmidiex/rmx/pkg/jobq"
	"github.com/redis/go-redis/v9"
	"github.com/zitadel/oidc/v2/pkg/client/rp"
	"github.com/zitadel/oidc/v2/pkg/oidc"
	"gocloud.dev/pubsub"
)

type Service struct {
	mux service.Service

	repo      authDB.Repo
	providers []*provider.Handlers

	BaseURI string
}

func New(ctx context.Context, providers []provider.Provider, opts ...Option) *Service {
	s := Service{
		mux: service.New(),
	}

	for _, opt := range opts {
		opt(&s)
	}

	if err := s.initProviders(providers); err != nil {
		log.Fatal(err)
	}

	/*
		if err := s.initQueue(ctx, qCap); err != nil {
			log.Fatal(err)
		}
	*/

	s.routes()
	return &s
}

func (s *Service) initProviders(providers []provider.Provider) error {
	var phs []*provider.Handlers
	for _, p := range providers {
		hs, err := p.Init(s.BaseURI, s.withCheckUser())
		if err != nil {
			return err
		}

		phs = append(phs, hs)
	}

	s.providers = phs
	return nil
}

func (s *Service) initQueue(ctx context.Context, subject string, cap int) error {
	_, err := jobq.New(ctx, subject)
	if err != nil {
		return err
	}

	if err := jobq.AsyncSubscribe(ctx, subject, s.introspect, 10); err != nil {
		return err
	}

	return nil
}

func (s *Service) introspect(m *pubsub.Message) {
	// rs.Introspect()
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Service) routes() {
	for _, p := range s.providers {
		s.mux.Handle(p.AuthURI, p.AuthHandler)
		s.mux.Handle(p.CallbackURI, p.CallbackHandler)
	}
}

// any idea what to name this?
const rtCookieName = "RMX_AUTH_RT"

func (s *Service) withCheckUser() rp.CodeExchangeUserinfoCallback[*oidc.IDTokenClaims] {
	type response struct {
		AccessToken string `json:"access_token"`
		IDToken     string `json:"id_token"`
	}

	return func(
		w http.ResponseWriter,
		r *http.Request,
		tokens *oidc.Tokens[*oidc.IDTokenClaims],
		state string,
		provider rp.RelyingParty,
		info *oidc.UserInfo,
	) {
		_, err := s.repo.GetUserByEmail(r.Context(), info.Email)
		if err != nil {
			if err == sql.ErrNoRows {
				// user does not exist, create a new one
				err := s.createUser(r.Context(), info)
				if err != nil {
					s.mux.Respond(w, r, err, http.StatusInternalServerError)
					return
				}

				_, err = s.repo.GetUserByEmail(r.Context(), info.Email)
				if err != nil {
					s.mux.Respond(w, r, err, http.StatusInternalServerError)
					return
				}
			} else {
				s.mux.Respond(w, r, err, http.StatusInternalServerError)
				return
			}
		}

		rtCookie := &http.Cookie{
			Name:     rtCookieName,
			Value:    tokens.RefreshToken,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
			Expires:  tokens.Expiry,
		}

		res := &response{
			AccessToken: tokens.AccessToken,
			IDToken:     tokens.IDToken,
		}

		http.SetCookie(w, rtCookie)
		s.mux.Respond(w, r, res, http.StatusOK)
	}
}

func (s *Service) createUser(ctx context.Context, info *oidc.UserInfo) error {
	user := auth.User{
		Username: info.GivenName,
		Email:    info.Email,
	}

	_, err := s.repo.CreateUser(ctx, user)
	return err
}

type Option func(*Service)

func WithRepo(conn *sql.DB, rc *redis.Client) Option {
	return func(s *Service) {
		s.repo = authDB.New(conn, rc)
	}
}

func WithBaseURI(uri string) Option {
	return func(s *Service) {
		s.BaseURI = uri
	}
}
