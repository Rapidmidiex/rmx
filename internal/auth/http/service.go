package http

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/nats-io/nats.go"
	"github.com/rapidmidiex/oauth"
	"github.com/rapidmidiex/rmx/internal/auth"
	authStore "github.com/rapidmidiex/rmx/internal/auth/store"
	"github.com/rapidmidiex/rmx/internal/cache"
	service "github.com/rapidmidiex/rmx/internal/http"
)

type Service struct {
	ctx         context.Context
	mux         service.Service
	client      *oauth.Client
	nc          *nats.Conn
	repo        authStore.Repo
	keyPair     *auth.KeyPair
	callbackURL string
	errc        chan error
}

func New(opts ...Option) *Service {
	s := Service{
		mux:    service.New(),
		client: oauth.NewClient(),
		errc:   make(chan error),
	}

	for _, opt := range opts {
		opt(&s)
	}

	s.routes(s.client.GetProviders())
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
	s.mux.Get("/{provider}", s.handleAuth())
	s.mux.Get("/{provider}/callback", s.handleCallback())
	s.mux.Get("/refresh", s.handleRefresh())
	// s.mux.Handle("/protected", middlewares.VerifySession(s.handleProtected(), s.nc, s.keyPair.PublicKey))
}

const sessionCookieName = "_rmx_oauth_session"

func (s *Service) handleAuth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		providerName := chi.URLParam(r, "provider")
		provider, err := s.client.GetProvider(providerName)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("provider not found"))
			return
		}

		sess, err := provider.BeginAuth(oauth.SetState(r))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("couldn't begin auth"))
			return
		}

		url, err := sess.GetAuthURL()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("couldn't get auth url"))
			return
		}

		str, err := sess.Marshal()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("couldn't get auth url"))
			return
		}

		sid, err := s.repo.SaveSession([]byte(str))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("couldn't get auth url"))
			return
		}

		cookie := &http.Cookie{
			Name:     sessionCookieName,
			Value:    sid,
			HttpOnly: true,
			Secure:   false,
			SameSite: http.SameSiteLaxMode,
			Expires:  time.Now().UTC().Add(auth.RefreshTokenExp),
		}

		http.SetCookie(w, cookie)
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	}
}

func (s *Service) handleCallback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		providerName := chi.URLParam(r, "provider")
		provider, err := s.client.GetProvider(providerName)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("provider not found"))
			return
		}

		sessBytes, err := s.getSession(r)
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("couldn't get session"))
			return
		}

		sess, err := provider.UnmarshalSession(string(sessBytes))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("couldn't unmarshal session"))
			return
		}

		if err := oauth.ValidateState(r, sess); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("couldn't validate session"))
			return
		}

		user, err := fetchUser(r, provider, sess)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("couldn't fetch user data"))
			return
		}

		if err := s.saveUser(r, &user); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("couldn't save user"))
			return
		}

		// accessToken, refreshToken, err := s.genTokens(provider.Name(), uuid.NewString(), user.Email)
		// if err != nil {
		// 	w.WriteHeader(http.StatusInternalServerError)
		// 	return
		// }

		// if err := setTokens(w, accessToken, refreshToken); err != nil {
		// 	w.WriteHeader(http.StatusInternalServerError)
		// 	return
		// }

		http.Redirect(w, r, s.callbackURL, http.StatusTemporaryRedirect)
	}
}

func (s *Service) getSession(r *http.Request) ([]byte, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil, err
	}

	return s.repo.GetSession(cookie.Value)
}

func fetchUser(r *http.Request, provider oauth.Provider, sess oauth.Session) (oauth.User, error) {
	user, err := provider.FetchUser(sess)
	if err == nil {
		return user, nil
	}

	params := r.URL.Query()
	if params.Encode() == "" && r.Method == http.MethodPost {
		r.ParseForm()
		params = r.Form
	}

	// get new token and retry fetch
	_, err = sess.Authorize(provider, params)
	if err != nil {
		return oauth.User{}, err
	}

	return provider.FetchUser(sess)
}

func (s *Service) saveUser(r *http.Request, user *oauth.User) error {
	_, err := s.repo.GetUserByEmail(r.Context(), user.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// user does not exist, create a new one
			if _, err := s.repo.CreateUser(r.Context(), &auth.User{
				Email:    user.Email,
				Username: user.Name,
			}); err != nil {
				return err
			}

			if _, err := s.repo.GetUserByEmail(r.Context(), user.Email); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}

func (s *Service) handleRefresh() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

func (s *Service) genTokens(issuer, sid, email string) (string, string, error) {
	accessToken, err := jwt.NewBuilder().
		JwtID(uuid.NewString()).
		Issuer(issuer).
		Audience([]string{"web"}).
		Subject(sid).
		IssuedAt(time.Now().UTC()).
		NotBefore(time.Now().UTC()).
		Expiration(time.Now().UTC().Add(auth.AccessTokenExp)).
		Claim("email", email).
		Build()
	if err != nil {
		return "", "", err
	}

	bs, err := json.Marshal(accessToken)
	if err != nil {
		return "", "", err
	}

	atSigned, err := jws.Sign(bs, jws.WithKey(jwa.ES256, s.keyPair.PrivateKey))
	if err != nil {
		return "", "", err
	}

	refreshToken, err := jwt.NewBuilder().
		JwtID(uuid.NewString()).
		Issuer(issuer).
		Audience([]string{"web"}).
		Subject(sid).
		IssuedAt(time.Now().UTC()).
		NotBefore(time.Now().UTC()).
		Expiration(time.Now().UTC().Add(auth.RefreshTokenExp)).
		Claim("email", email).
		Build()
	if err != nil {
		return "", "", err
	}

	bs, err = json.Marshal(refreshToken)
	if err != nil {
		return "", "", err
	}

	rtSigned, err := jws.Sign(bs, jws.WithKey(jwa.ES256, s.keyPair.PrivateKey))
	if err != nil {
		return "", "", err
	}

	return string(atSigned), string(rtSigned), nil
}

// func setTokens(w http.ResponseWriter, accessToken, refreshToken string) string {
// 	type response struct {
// 		AccessToken string `json:"accessToken"`
// 	}

// 	cookie := &http.Cookie{
// 		Name:     sessionCookieName,
// 		Value:    refreshToken,
// 		Expires:  time.Now().UTC().Add(auth.RefreshTokenExp),
// 		Secure:   true,
// 		HttpOnly: true,
// 		SameSite: http.SameSiteLaxMode,
// 	}

// 	http.SetCookie(w, cookie)
// 	return ""
// }

func (s *Service) handleProtected() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {}
}

type Option func(*Service)

func WithContext(ctx context.Context) Option {
	return func(s *Service) {
		s.ctx = ctx
	}
}

func WithKeys(privk *ecdsa.PrivateKey, pubk *ecdsa.PublicKey) Option {
	return func(s *Service) {
		s.keyPair = &auth.KeyPair{
			PrivateKey: privk,
			PublicKey:  pubk,
		}
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
		s.client.UseProviders(p...)
	}
}

func WithCallback(callbackURL string) Option {
	return func(s *Service) {
		s.callbackURL = callbackURL
	}
}
