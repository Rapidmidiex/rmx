package http

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/rapidmidiex/rmx/internal/auth"
	authDB "github.com/rapidmidiex/rmx/internal/auth/postgres"
	"github.com/rapidmidiex/rmx/internal/auth/provider"
	"github.com/rapidmidiex/rmx/internal/auth/provider/google"
	service "github.com/rapidmidiex/rmx/internal/http"
	"github.com/zitadel/oidc/v2/pkg/client/rp"
	"github.com/zitadel/oidc/v2/pkg/oidc"
)

type Service struct {
	mux service.Service

	repo      authDB.Repo
	providers []*auth.Provider
}

func New(ctx context.Context, baseURI string, repo authDB.Repo) *Service {
	s := Service{
		mux: service.New(),

		repo: repo,
	}

	if err := s.initProviders(ctx, baseURI); err != nil {
		log.Fatal(err)
	}
	s.routes()
	return &s
}

func (s *Service) initProviders(ctx context.Context, baseURI string) error {
	googleCfg := &provider.ProviderCfg{
		BaseURI: baseURI,
	}
	google, err := google.New(googleCfg, s.checkUser(ctx))
	if err != nil {
		return err
	}

	s.providers = append(s.providers, google)
	return nil
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

func (s *Service) checkUser(ctx context.Context) rp.CodeExchangeUserinfoCallback[*oidc.IDTokenClaims] {
	return func(
		w http.ResponseWriter,
		r *http.Request,
		tokens *oidc.Tokens[*oidc.IDTokenClaims],
		state string,
		provider rp.RelyingParty,
		info *oidc.UserInfo,
	) {
		userInfo, err := s.repo.GetUserByEmail(ctx, info.Email)
		if err != nil {
			if err == sql.ErrNoRows {
				s.createUser()
				return
			}

			s.mux.Respond(w, r, err, http.StatusInternalServerError)
			return
		}

		bs, err := json.Marshal(userInfo)
		if err != nil {
			s.mux.Respond(w, r, err, http.StatusInternalServerError)
			return
		}

		s.mux.Respond(w, r, bs, http.StatusOK)
	}
}

func (s *Service) createUser() {
	// not implemented yet
}
