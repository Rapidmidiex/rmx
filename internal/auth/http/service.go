package http

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/hyphengolang/prelude/types/suid"
	"github.com/rapidmidiex/rmx/internal/auth"
	"github.com/rapidmidiex/rmx/internal/auth/internal/token"
	authDB "github.com/rapidmidiex/rmx/internal/auth/postgres"
	"github.com/rapidmidiex/rmx/internal/auth/provider"
	"github.com/rapidmidiex/rmx/internal/cache"
	service "github.com/rapidmidiex/rmx/internal/http"
	"github.com/zitadel/oidc/v2/pkg/client/rp"
	"github.com/zitadel/oidc/v2/pkg/oidc"
	"gocloud.dev/pubsub"
)

type Service struct {
	mux service.Service

	repo      authDB.Repo
	providers []*provider.Handlers
	baseURI   string
}

func New(ctx context.Context, opts ...Option) *Service {
	s := Service{
		mux: service.New(),
	}

	for _, opt := range opts {
		opt(&s)
	}

	s.routes()
	return &s
}

func (s *Service) GetBaseURI() string { return s.baseURI }

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

type Option func(*Service)

func WithRepo(conn *sql.DB, cache *cache.Cache) Option {
	return func(s *Service) {
		s.repo = authDB.New(conn, cache)
	}
}

func WithBaseURI(uri string) Option {
	return func(s *Service) {
		s.baseURI = uri
	}
}

func WithProviders(providers []provider.Provider, pk *ecdsa.PrivateKey) Option {
	return func(s *Service) {
		for _, p := range providers {
			hs, err := p.Init(s.baseURI, s.withCheckUser(pk))
			if err != nil {
				log.Fatal(err)
			}

			s.providers = append(s.providers, hs)
		}
	}
}

// any idea what to name this?
const rtCookieName = "RMX_AUTH_RT"

func (s *Service) withCheckUser(pk *ecdsa.PrivateKey) rp.CodeExchangeUserinfoCallback[*oidc.IDTokenClaims] {
	type response struct {
		AccessToken string `json:"accessToken"`
		IDToken     string `json:"idToken"`
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

		cid := suid.NewSUID().String()
		if err := s.createSession(cid, provider.Issuer(), info, tokens); err != nil {
			s.mux.Respond(w, r, err, http.StatusInternalServerError)
			return
		}

		at, err := token.New(&token.Claims{
			Issuer:     provider.Issuer(),
			Audience:   []string{"web"}, // TODO: choose audience
			Email:      info.Email,
			ClientID:   cid,
			Expiration: tokens.Expiry,
		}, pk)
		if err != nil {
			s.mux.Respond(w, r, err, http.StatusInternalServerError)
			return
		}

		rt, err := token.New(&token.Claims{
			Issuer:     provider.Issuer(),
			Audience:   []string{"web"},
			Email:      info.Email,
			ClientID:   cid,
			Expiration: time.Now().UTC().Add(time.Hour * 24 * 30), // a month
		}, pk)
		if err != nil {
			s.mux.Respond(w, r, err, http.StatusInternalServerError)
			return
		}

		rtCookie := &http.Cookie{
			Name:     rtCookieName,
			Value:    rt,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
			Expires:  tokens.Expiry,
		}

		res := &response{
			AccessToken: at,
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

func (s *Service) createSession(
	cid string,
	issuer string,
	info *oidc.UserInfo,
	tokens *oidc.Tokens[*oidc.IDTokenClaims],
) error {
	return s.repo.CreateSession(
		info.Email,
		issuer,
		cid,
		auth.Tokens{
			AccessToken:  tokens.AccessToken,
			RefreshToken: tokens.RefreshToken,
		},
	)
}
