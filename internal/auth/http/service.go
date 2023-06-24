package http

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/nats-io/nats.go"
	"github.com/rapidmidiex/oauth"
	"github.com/rapidmidiex/oauth/auther"
	"github.com/rapidmidiex/rmx/internal/auth"
	"github.com/rapidmidiex/rmx/internal/auth/internal/token"
	authStore "github.com/rapidmidiex/rmx/internal/auth/store"
	"github.com/rapidmidiex/rmx/internal/cache"
	service "github.com/rapidmidiex/rmx/internal/http"
	"github.com/rapidmidiex/rmx/internal/middlewares"
	"github.com/zitadel/oidc/v2/pkg/oidc"
)

type Service struct {
	ctx     context.Context
	mux     service.Service
	nc      *nats.Conn
	repo    authStore.Repo
	keyPair auth.KeyPair
	errc    chan error
}

func New(ctx context.Context, opts ...Option) *Service {
	s := Service{
		mux:  service.New(),
		errc: make(chan error),
	}

	for _, opt := range opts {
		opt(&s)
	}

	s.routes(oauth.GetProviders())
	go s.errors()
	return &s
}

// I have no idea what to do with the errors here
func (s *Service) errors() {
	for {
		err := <-s.errc
		log.Println(err.Error())
	}
}

// func (s *Service) introspect() {
// 	subj := fmt.Sprint(events.NatsSubj, events.NatsSessionSufx, events.NatsIntrospectSufx)
// 	if _, err := s.nc.Subscribe(subj, func(msg *nats.Msg) {
// 		at := string(msg.Data)
// 		parsed, err := jwt.Parse([]byte(at), jwt.WithKey(jwa.ES256, s.keyPair.PublicKey))
// 		if err != nil {
// 			if err := msg.Respond([]byte(events.TokenRejected)); err != nil {
// 				s.errc <- fmt.Errorf("rmx: introspect [parse]\n%v", err)
// 			}
// 		}

// 		res, err := s.repo.VerifyToken(s.ctx, parsed.JwtID())
// 		if err != nil {
// 			s.errc <- fmt.Errorf("rmx: introspect [verify]\n%v", err)
// 		}

// 		if err := msg.Respond([]byte(res)); err != nil {
// 			s.errc <- fmt.Errorf("rmx: introspect [result]\n%v", err)
// 		}
// 	}); err != nil {
// 		log.Fatalf("rmx: introspect\n%v", err)
// 	}
// }

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Service) routes(ps oauth.Providers) {
	s.mux.Get("/{provider}", func(w http.ResponseWriter, r *http.Request) {
		providerName := chi.URLParam(r, "provider")
		provider, err := oauth.GetProvider(providerName)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		sess, err := provider.BeginAuth(auther.SetState(r))
		if err != nil {
			fmt.Println("here")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		url, err := sess.GetAuthURL()
		if err != nil {
			fmt.Println("here2")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		fmt.Print(sess.Marshal() + "\n")
		cookie := &http.Cookie{
			Name:     "_rmx_session",
			Value:    sess.Marshal(),
			Path:     "/",
			Expires:  time.Now().UTC().Add(auth.RefreshTokenExp),
			Secure:   false,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		}

		http.SetCookie(w, cookie)
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	})
	s.mux.Get("/{provider}/callback", func(w http.ResponseWriter, r *http.Request) {
		providerName := chi.URLParam(r, "provider")
		provider, err := oauth.GetProvider(providerName)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		cookie, err := r.Cookie("_rmx_session")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		sess, err := provider.UnmarshalSession(cookie.Value)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		defer auther.Logout(w, r)

		err = validateState(r, sess)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		user, err := provider.FetchUser(sess)
		if err == nil {
			fmt.Println(user)
			return
		}

		params := r.URL.Query()
		if params.Encode() == "" && r.Method == "POST" {
			r.ParseForm()
			params = r.Form
		}

		// get new token and retry fetch
		_, err = sess.Authorize(provider, params)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		gu, err := provider.FetchUser(sess)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		fmt.Printf("%+v\n", gu)
	})
	s.mux.Get("/refresh", func(w http.ResponseWriter, r *http.Request) {
	})
	// s.mux.Handle("/protected", middlewares.VerifySession(s.handleProtected(), s.nc, s.keyPair.PublicKey))
}

// validateState ensures that the state token param from the original
// AuthURL matches the one included in the current (callback) request.
func validateState(req *http.Request, sess oauth.Session) error {
	rawAuthURL, err := sess.GetAuthURL()
	if err != nil {
		return err
	}

	authURL, err := url.Parse(rawAuthURL)
	if err != nil {
		return err
	}

	reqState := auther.GetState(req)

	originalState := authURL.Query().Get("state")
	if originalState != "" && (originalState != reqState) {
		return errors.New("state token mismatch")
	}
	return nil
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

// _, err := s.repo.GetUserByEmail(r.Context(), info.Email)
// if err != nil {
// 	if err == sql.ErrNoRows {
// 		// user does not exist, create a new one
// 		err := s.createUser(r.Context(), info)
// 		if err != nil {
// 			s.mux.Respond(w, r, err, http.StatusInternalServerError)
// 			return
// 		}

// 		_, err = s.repo.GetUserByEmail(r.Context(), info.Email)
// 		if err != nil {
// 			s.mux.Respond(w, r, err, http.StatusInternalServerError)
// 			return
// 		}
// 	} else {
// 		s.mux.Respond(w, r, err, http.StatusInternalServerError)
// 		return
// 	}
// }

type Option func(*Service)

func WithContext(ctx context.Context) Option {
	return func(s *Service) {
		s.ctx = ctx
	}
}

func WithKeys(privk *ecdsa.PrivateKey, pubk *ecdsa.PublicKey) Option {
	return func(s *Service) {
		s.keyPair.PrivateKey = privk
		s.keyPair.PublicKey = pubk
	}
}

func WithNats(nc *nats.Conn) Option {
	return func(s *Service) {
		s.nc = nc
	}
}

func WithRepo(dbConn *sql.DB, sessionCache, tokensCache *cache.Cache) Option {
	return func(s *Service) {
		s.repo = authStore.New(dbConn, sessionCache, tokensCache)
	}
}

func WithProviders(p ...oauth.Provider) Option {
	return func(s *Service) {
		oauth.UseProviders(p...)
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
