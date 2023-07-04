package http

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
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
}

func New(opts ...Option) *Service {
	s := Service{
		mux:    service.New(),
		client: oauth.NewClient(),
	}

	for _, opt := range opts {
		opt(&s)
	}

	s.routes(s.client.GetProviders())
	return &s
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Service) routes(ps oauth.Providers) {
	s.mux.Get("/{provider}", s.handleAuth())
	s.mux.Get("/{provider}/callback", s.handleCallback())
	// s.mux.Get("/refresh", s.handleRefresh())
	s.mux.Handle("/protected", auth.VerifySession(s.handleProtected(), s.keyPair.PublicKey, true))
}

func (s *Service) handleProtected() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, err := auth.GetSessionFromContext(r.Context())
		if err != nil {
			w.Write([]byte("you are unauthorized"))
			return
		}

		w.Write([]byte(fmt.Sprint(sess.SID, " ", sess.Issuer, " ", sess.Email)))
		return
	}
}

const (
	sessionCookieName = "_rmx_session"
	oauthCookieName   = "_rmx_oauth"
)

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

		if err := s.setOAuthSession(w, sess, provider.Name()); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("couldn't set session"))
			return
		}

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

		sessCookie, err := r.Cookie(oauthCookieName)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("no session cookie"))
			return
		}

		sess, err := s.getOAuthSession(sessCookie.Value, provider)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("no session found"))
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

		accessToken, err := s.newToken(provider.Name(), sessCookie.Value, user.Email, auth.AccessTokenExp)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("couldn't generate access token"))
			return
		}

		refreshToken, err := s.newToken(provider.Name(), sessCookie.Value, user.Email, auth.RefreshTokenExp)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("couldn't generate refresh token"))
			return
		}

		// this is so stupid and dangerous, tokens will expire tho
		callbackURL, err := url.Parse(s.callbackURL)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("couldn't parse callback url"))
			return
		}
		q := callbackURL.Query()
		q.Set("accessToken", accessToken)
		callbackURL.RawQuery = q.Encode()

		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookieName,
			Value:    refreshToken,
			Path:     "/",
			MaxAge:   int(auth.RefreshTokenExp.Seconds()), // 30 days
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
		})

		http.Redirect(w, r, callbackURL.String(), http.StatusTemporaryRedirect)
	}
}

func (s *Service) setOAuthSession(w http.ResponseWriter, sess oauth.Session, providerName string) error {
	sid, err := s.repo.SaveSession(sess)
	if err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     oauthCookieName,
		Value:    sid,
		Path:     "/",
		MaxAge:   60 * 5, // 5 minutes
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	return nil
}

func (s *Service) getOAuthSession(sid string, provider oauth.Provider) (oauth.Session, error) {
	sess, err := s.repo.GetSession(sid)
	if err != nil {
		return nil, err
	}

	return provider.UnmarshalSession(string(sess))
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
	type response struct {
		AccessToken string `json:"accessToken"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		parsed, err := s.parseToken(cookie.Value)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// TODO: verify oauth session
		// check if session is not blacklisted
		ok, err := s.repo.VerifySession(r.Context(), parsed.JwtID())
		if !ok || err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		//	bs, err := s.repo.GetSession(parsed.JwtID())
		//	if err != nil {
		//		w.WriteHeader(http.StatusUnauthorized)
		//		return
		//	}

		provider, err := s.client.GetProvider(parsed.Issuer())
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		//	sess, err := provider.UnmarshalSession(string(bs))
		//	if err != nil {
		//		w.WriteHeader(http.StatusUnauthorized)
		//		return
		//	}

		//	sess.VerifySession() // not implemented yet

		accessToken, err := s.newToken(provider.Name(), parsed.JwtID(), parsed.Subject(), auth.AccessTokenExp)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// subtract the refresh token expiry elapsed time
		exp := auth.RefreshTokenExp - time.Now().UTC().Sub(parsed.Expiration())
		refreshToken, err := s.newToken(provider.Name(), parsed.JwtID(), parsed.Subject(), exp)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("couldn't generate refresh token"))
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookieName,
			Value:    refreshToken,
			Path:     "/",
			MaxAge:   int(exp.Seconds()),
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
		})

		s.mux.Respond(w, r, &response{accessToken}, http.StatusOK)
	}
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

func WithCallbackURL(callbackURL string) Option {
	return func(s *Service) {
		s.callbackURL = callbackURL
	}
}
