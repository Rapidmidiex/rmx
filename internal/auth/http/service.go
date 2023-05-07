package http

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	authDB "github.com/rapidmidiex/rmx/internal/auth/postgres"
	"github.com/rapidmidiex/rmx/internal/auth/provider"
	service "github.com/rapidmidiex/rmx/internal/http"
	"github.com/zitadel/oidc/v2/pkg/client/rp"
	"github.com/zitadel/oidc/v2/pkg/oidc"
)

type Service struct {
	mux service.Service

	repo      authDB.Repo
	providers []*provider.Handlers
	BaseURI   string
}

func New(ctx context.Context, baseURI string, repo authDB.Repo, providers []provider.Provider) *Service {
	s := Service{
		mux: service.New(),

		repo:    repo,
		BaseURI: baseURI,
	}

	if err := s.initProviders(ctx, baseURI, providers); err != nil {
		log.Fatal(err)
	}
	s.routes()
	return &s
}

func (s *Service) initProviders(ctx context.Context, baseURI string, providers []provider.Provider) error {
	var phs []*provider.Handlers
	for _, p := range providers {
		hs, err := p.Init(baseURI, s.checkUser(ctx))
		if err != nil {
			return err
		}

		phs = append(phs, hs)
	}

	s.providers = phs
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
