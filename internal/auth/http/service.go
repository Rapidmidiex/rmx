package http

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/nats-io/nats.go"
	"github.com/rapidmidiex/rmx/internal/auth"
	"github.com/rapidmidiex/rmx/internal/auth/internal/token"
	"github.com/rapidmidiex/rmx/internal/auth/provider"
	authStore "github.com/rapidmidiex/rmx/internal/auth/store"
	"github.com/rapidmidiex/rmx/internal/cache"
	"github.com/rapidmidiex/rmx/internal/events"
	service "github.com/rapidmidiex/rmx/internal/http"
	"github.com/rapidmidiex/rmx/internal/middlewares"
	"github.com/zitadel/oidc/v2/pkg/client/rp"
	"github.com/zitadel/oidc/v2/pkg/oidc"
)

type Service struct {
	ctx       context.Context
	mux       service.Service
	nc        *nats.Conn
	repo      authStore.Repo
	providers []*provider.Handlers
	baseURI   string
	pubk      *ecdsa.PublicKey
	errc      chan error
}

func New(ctx context.Context, opts ...Option) *Service {
	s := Service{
		mux:  service.New(),
		errc: make(chan error),
	}

	for _, opt := range opts {
		opt(&s)
	}

	s.routes()
	go s.errors()
	go s.introspect()
	return &s
}

func (s *Service) GetBaseURI() string { return s.baseURI }

// I have no idea what to do with the errors here
func (s *Service) errors() {
	for {
		err := <-s.errc
		log.Println(err.Error())
	}
}

func (s *Service) introspect() {
	subj := fmt.Sprint(events.NatsSubj, events.NatsSessionSufx, events.NatsIntrospectSufx)
	s.nc.Subscribe(subj, func(msg *nats.Msg) {
		token := string(msg.Data)
		parsed, err := jwt.Parse([]byte(token), jwt.WithKey(jwa.ES256, s.pubk))
		if err != nil {
			if err := msg.Respond([]byte(events.TokenRejected)); err != nil {
				s.errc <- fmt.Errorf("rmx: introspect [parse]\n%v", err)
			}
		}

		res, err := s.repo.VerifyToken(s.ctx, parsed.JwtID())
		if err != nil {
			s.errc <- fmt.Errorf("rmx: introspect [verify]\n%v", err)
		}

		if err := msg.Respond([]byte(res)); err != nil {
			s.errc <- fmt.Errorf("rmx: introspect [result]\n%v", err)
		}
	})
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Service) routes() {
	for _, p := range s.providers {
		s.mux.Handle(p.AuthURI, p.AuthHandler)
		s.mux.Handle(p.CallbackURI, p.CallbackHandler)
	}
	s.mux.Handle("/refresh", s.handleRefresh())
	s.mux.Handle("/protected", middlewares.ParseSession(middlewares.VerifySession(s.handleProtected(), s.nc)))
}

func (s *Service) handleProtected() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("welcome to RMX!"))
	}
}

func (s *Service) handleRefresh() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

type Option func(*Service)

func WithContext(ctx context.Context) Option {
	return func(s *Service) {
		s.ctx = ctx
	}
}

func WithNats(conn *nats.Conn) Option {
	return func(s *Service) {
		s.nc = conn
	}
}

func WithPublicKey(pk *ecdsa.PublicKey) Option {
	return func(s *Service) {
		s.pubk = pk
	}
}

func WithRepo(conn *sql.DB, sessionCache, tokenCache *cache.Cache) Option {
	return func(s *Service) {
		s.repo = authStore.New(conn, sessionCache, tokenCache)
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

		cid, err := s.createSession(s.ctx, provider.Issuer(), info, tokens)
		if err != nil {
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
			Expiration: time.Now().UTC().Add(auth.RefreshTokenExp),
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
	ctx context.Context,
	issuer string,
	info *oidc.UserInfo,
	tokens *oidc.Tokens[*oidc.IDTokenClaims],
) (string, error) {
	return s.repo.CreateSession(
		ctx,
		info.Email,
		issuer,
		auth.Session{
			AccessToken:  tokens.AccessToken,
			RefreshToken: tokens.RefreshToken,
		},
	)
}
