package http

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"net/url"

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
	// s.mux.Get("/refresh", s.handleRefresh())
	// s.mux.Handle("/protected", middlewares.VerifySession(s.handleProtected(), s.nc, s.keyPair.PublicKey))
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

		str, err := sess.Marshal()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("couldn't marshal session"))
			return
		}

		sid, err := s.repo.SaveSession(&auth.Session{
			Provider:    provider.Name(),
			SessionInfo: str,
		})
		println(sid)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("couldn't save session"))
			return
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

		appSession, err := s.repo.GetSession(sessCookie.Value)
		if err != nil {
			println(err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("couldn't find session"))
			return
		}

		sess, err := provider.UnmarshalSession(string(appSession.SessionInfo))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
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

		accessToken, refreshToken, err := s.genTokens(provider.Name(), sessCookie.Value, user.Email)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("couldn't set session"))
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

		cookie := &http.Cookie{
			Name:     sessionCookieName,
			Value:    refreshToken,
			Path:     "/",
			MaxAge:   86400 * 30, // 30 days
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
		}

		http.SetCookie(w, cookie)
		http.Redirect(w, r, callbackURL.String(), http.StatusTemporaryRedirect)
	}
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

// func (s *Service) handleRefresh() http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		cookie, _ := r.Cookie(sessionCookieName)
// 		sess, _ := s.repo.GetSession(cookie.Value)
// 		provider, _ := s.client.GetProvider(sess.Provider)
// 		sessInfo, _ := provider.UnmarshalSession(sess.SessionInfo)
// 		provider.
// 	}
// }

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
