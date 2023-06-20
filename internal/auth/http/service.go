package http

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"errors"
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
	providers map[string]provider.Provider
	baseURI   string
	pubk      *ecdsa.PublicKey
	privk     *ecdsa.PrivateKey
	errc      chan error
}

func New(ctx context.Context, opts ...Option) *Service {
	s := Service{
		mux:       service.New(),
		providers: make(map[string]provider.Provider),
		errc:      make(chan error),
	}

	for _, opt := range opts {
		opt(&s)
	}

	var phs []*provider.Handlers
	for _, p := range s.providers {
		hs, err := p.GetHandlers(s.baseURI, s.withCheckUser(s.privk))
		if err != nil {
			log.Fatal(err)
		}

		phs = append(phs, hs)
	}

	s.routes(phs)
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
	if _, err := s.nc.Subscribe(subj, func(msg *nats.Msg) {
		at := string(msg.Data)
		parsed, err := jwt.Parse([]byte(at), jwt.WithKey(jwa.ES256, s.pubk))
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
	}); err != nil {
		log.Fatalf("rmx: introspect\n%v", err)
	}
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Service) routes(phs []*provider.Handlers) {
	for _, p := range phs {
		s.mux.Handle(p.AuthURI, p.AuthHandler)
		s.mux.Handle(p.CallbackURI, p.CallbackHandler)
	}
	s.mux.Handle("/refresh", s.handleRefresh())
	s.mux.Handle("/protected", middlewares.VerifySession(s.handleProtected(), s.nc, s.pubk))
}

func (s *Service) handleProtected() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, ok := r.Context().Value(middlewares.SessionCtx).(token.ParsedClaims)
		if !ok {
			s.mux.Respond(w, r, nil, http.StatusBadRequest)
			return
		}

		s.mux.Respond(w, r, session, http.StatusOK)
	}
}

func (s *Service) handleRefresh() http.HandlerFunc {
	type response struct {
		AccessToken string `json:"accessToken"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		rtCookie, err := r.Cookie(auth.RefreshTokenCookieName)
		if err != nil {
			s.mux.Respond(w, r, nil, http.StatusBadRequest)
			return
		}

		rt := token.TrimPrefix(rtCookie.Value)

		parsed, err := jwt.Parse([]byte(rt), jwt.WithKey(jwa.ES256, s.pubk))
		if err != nil {
			s.mux.Respond(w, r, nil, http.StatusUnauthorized)
			return
		}

		session, err := s.repo.GetSession(parsed.Subject())
		if err != nil {
			s.mux.Respond(w, r, nil, http.StatusUnauthorized)
			return
		}

		// no need for the response here
		_, err = s.validateSession(parsed.Issuer(), session)
		if err != nil {
			s.mux.Respond(w, r, nil, http.StatusUnauthorized)
			return
		}

		emailInterface, ok := parsed.Get("email")
		if !ok {
			s.mux.Respond(w, r, nil, http.StatusBadRequest)
			return
		}

		email, ok := emailInterface.(string)
		if !ok {
			s.mux.Respond(w, r, nil, http.StatusBadRequest)
			return
		}

		at, err := token.New(&token.Claims{
			Issuer:     parsed.Issuer(),
			Audience:   parsed.Audience(), // TODO: choose audience
			Email:      email,
			ClientID:   parsed.Subject(),
			Expiration: time.Now().UTC().Add(auth.AccessTokenExp),
		}, s.privk)
		if err != nil {
			s.mux.Respond(w, r, err, http.StatusInternalServerError)
			return
		}

		newRt, err := token.New(&token.Claims{
			Issuer:     parsed.Issuer(),
			Audience:   parsed.Audience(), // TODO: choose audience
			Email:      email,
			ClientID:   parsed.Subject(),
			Expiration: time.Now().UTC().Add(auth.RefreshTokenExp),
		}, s.privk)
		if err != nil {
			s.mux.Respond(w, r, err, http.StatusInternalServerError)
			return
		}

		newRtCookie := &http.Cookie{
			Name:     auth.RefreshTokenCookieName,
			Value:    newRt,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
			Expires:  time.Now().UTC().Add(auth.RefreshTokenExp),
		}

		res := &response{
			AccessToken: at,
		}

		http.SetCookie(w, newRtCookie)
		s.mux.Respond(w, r, res, http.StatusOK)
	}
}

func (s *Service) validateSession(issuer string, session *auth.Session) (*oidc.IntrospectionResponse, error) {
	// TODO: refresh session access token
	pr, ok := s.providers[issuer]
	if !ok {
		return nil, errors.New("rmx: invalid provider")
	}

	return pr.Introspect(s.ctx, session)
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

func WithKeys(privk *ecdsa.PrivateKey, pubk *ecdsa.PublicKey) Option {
	return func(s *Service) {
		s.privk = privk
		s.pubk = pubk
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

func WithProvider(provider provider.Provider) Option {
	return func(s *Service) {
		s.providers[provider.Issuer()] = provider
	}
}

func (s *Service) withCheckUser(pk *ecdsa.PrivateKey) rp.CodeExchangeCallback[*oidc.IDTokenClaims] {
	type response struct {
		AccessToken string `json:"accessToken"`
		IDToken     string `json:"idToken"`
	}

	return rp.UserinfoCallback[*oidc.IDTokenClaims](func(
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

		sid, err := s.createSession(s.ctx, provider.Issuer(), info, tokens)
		if err != nil {
			s.mux.Respond(w, r, err, http.StatusInternalServerError)
			return
		}

		at, err := token.New(&token.Claims{
			Issuer:     provider.Issuer(),
			Audience:   []string{"web"}, // TODO: choose audience
			Email:      info.Email,
			ClientID:   sid,
			Expiration: time.Now().UTC().Add(auth.AccessTokenExp),
		}, pk)
		if err != nil {
			s.mux.Respond(w, r, err, http.StatusInternalServerError)
			return
		}

		rt, err := token.New(&token.Claims{
			Issuer:     provider.Issuer(),
			Audience:   []string{"web"},
			Email:      info.Email,
			ClientID:   sid,
			Expiration: time.Now().UTC().Add(auth.RefreshTokenExp),
		}, pk)
		if err != nil {
			s.mux.Respond(w, r, err, http.StatusInternalServerError)
			return
		}

		rtCookie := &http.Cookie{
			Name:     auth.RefreshTokenCookieName,
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
	})
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
			TokenType:    tokens.TokenType,
			AccessToken:  tokens.AccessToken,
			RefreshToken: tokens.RefreshToken,
			Expiry:       tokens.Expiry,
		},
	)
}
