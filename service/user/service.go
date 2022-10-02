package user

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"

	h "github.com/hyphengolang/prelude/http"

	"github.com/rog-golang-buddies/rmx/internal"
	"github.com/rog-golang-buddies/rmx/internal/auth"
	"github.com/rog-golang-buddies/rmx/internal/suid"
	"github.com/rog-golang-buddies/rmx/test/mock"
)

// TODO use os/viper to get `key.pem` body
var secretTest = `-----BEGIN PRIVATE KEY-----
MIICdgIBADANBgkqhkiG9w0BAQEFAASCAmAwggJcAgEAAoGBAML5MHFgqUlZcENS
hHZ83yXfoUpqaMfp5/UdgMIJ0S5DW5QEON6reAsDu6zP0BEVZhg65pEYWEraBrGK
Vcbx7dsVqK4Z0GMm0YRAvB+1K+pYlXwld90mwG1TqOKDPQXqC0Z/jZi6DSsAhfJU
WN0rkInZRtoVeRzbbh+nLN8nd14fAgMBAAECgYEAor+A2VL3XBvFIt0RZxpq5mFa
cBSMrDsqfSeIX+/z5SsimVZA5lW5GXCfSuwY4Pm8xAL+jSUGJk0CA1bWrP8rLByS
cQAy1q0odaAiWIG5zFUEQBg5Q5b3+jXmh2zwtO7yhPuXn1/vBGg+FvyR57gV/3F+
TuBfR6Bc3VWKuj7Gm5kCQQDuRgm8HTDbX7IQ0EFAVKB73Pj4Gx5u2NieD9U8+qXx
JsAdn1vRvQ3mNJDR5OcTr4rPkpLLCtzjA2iTDXp4yqmrAkEA0Xp91LCpImKAOtM3
4SGXdzKi9+7fWmcTtfkz996y9A1C9l27Cj92P7OFdwMB4Z/ZMizJd0eXYhXr4IxH
wBoxXQJAUBOXp/HDfqZdiIsEsuL+AEKWJYOvqZ8UxaIajuDJrg7Q1+O7jvRTXH9k
ADZGdnYzV2kyDiy7aUu29Fy+QSQS+wJAJyEsdBhz35pqvZJK8+DkfD2XN50FV8u9
YNamIH0XDIOVqJOlpqpoGkocejizl0PWvIqlL4TOAGJ75zwNAxNheQJABEA2/hfF
GMJsOrnD74rGP/Lfpg882AmeUoT5eH766sSobFfUYJZvyAmnQoK2Lzg2hrKwXXix
JvEGfrhihVLb7g==
-----END PRIVATE KEY-----
`

type contextKey string

var emailKey = contextKey("rmx-email")

var (
	ErrNoCookie        = errors.New("user: cookie not found")
	ErrSessionNotFound = errors.New("user: session not found")
	ErrSessionExists   = errors.New("user: session already exists")
)

type Service struct {
	m  chi.Router
	ur internal.UserRepo

	l *log.Logger

	ac *auth.Client
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.m.ServeHTTP(w, r) }

func NewService(m chi.Router, r internal.UserRepo) *Service {
	s := &Service{m: m, ur: r, l: log.Default()}
	s.routes()
	return s
}

func DefaultService() *Service {
	s := &Service{m: chi.NewMux(), ur: mock.UserRepo(), l: log.Default()}
	s.routes()
	return s
}

func (s *Service) respond(w http.ResponseWriter, r *http.Request, data any, status int) {
	h.Respond(w, r, data, status)
}

func (s *Service) respondCookie(w http.ResponseWriter, r *http.Request, data any, c *http.Cookie) {
	http.SetCookie(w, c)
	s.respond(w, r, data, http.StatusOK)
}

func (s *Service) created(w http.ResponseWriter, r *http.Request, id string) {
	h.Created(w, r, id)
}

func (s *Service) decode(w http.ResponseWriter, r *http.Request, data interface{}) error {
	return h.Decode(w, r, data)
}

func (s *Service) parseUUID(w http.ResponseWriter, r *http.Request) (suid.UUID, error) {
	return suid.ParseString(chi.URLParam(r, "uuid"))
}

func (s *Service) signedTokens(key jwk.Key, now time.Time, email, uuid string) (its, ats, rts []byte, err error) {
	var jwb = jwt.NewBuilder().Issuer("github.com/rog-golang-buddies/rmx").IssuedAt(now).Claim("email", email)
	// Audience([]string{"http://localhost:3000"}).

	it, err := jwb.Subject(email).Expiration(now.Add(time.Hour * 10)).Build()
	if err != nil {
		return nil, nil, nil, err

	}

	its, err = jwt.Sign(it, jwt.WithKey(jwa.RS256, key))
	if err != nil {
		return nil, nil, nil, err

	}

	at, err := jwb.Subject(uuid).Expiration(now.Add(time.Minute * 5)).Build()
	if err != nil {
		return nil, nil, nil, err
	}

	ats, err = jwt.Sign(at, jwt.WithKey(jwa.RS256, key))
	if err != nil {
		return nil, nil, nil, err

	}

	rt, err := jwb.Subject(uuid).Expiration(now.Add(time.Hour * 24 * 7)).Build()
	if err != nil {
		return nil, nil, nil, err

	}

	rts, err = jwt.Sign(rt, jwt.WithKey(jwa.RS256, key))
	if err != nil {
		return nil, nil, nil, err
	}

	return its, ats, rts, nil
}

func (s *Service) routes() {
	// panic should be ok as we need this to return no error
	// else it'll completely break our auth model
	private, err := jwk.ParseKey([]byte(secretTest), jwk.WithPEM(true))
	if err != nil {
		panic(err)
	}

	public, err := private.PublicKey()
	if err != nil {
		panic(err)
	}

	s.m.Route("/api/v1/user", func(r chi.Router) {
		r.Post("/", s.handleRegistration())

		// health
		r.Get("/ping", s.handlePing)
	})

	s.m.Route("/api/v1/auth", func(r chi.Router) {
		r.Post("/login", s.handleLogin(private))

		auth := r.With(s.authenticate(public))
		auth.Get("/me", s.handleIdentity())
		auth.Get("/refresh", s.handleRefresh(private))
		auth.Post("/logout", s.handleLogout())
	})
}

func (s *Service) handlePing(w http.ResponseWriter, r *http.Request) {
	s.respond(w, r, nil, http.StatusNoContent)
}

type SignupUser struct {
	Email    internal.Email    `json:"email"`
	Username string            `json:"username"`
	Password internal.Password `json:"password"`
}

func (v SignupUser) decode(iu *internal.User) error {
	h, err := v.Password.Hash()
	if err != nil {
		return err
	}

	*iu = internal.User{
		ID:       suid.NewUUID(),
		Email:    v.Email,
		Username: v.Username,
		Password: h,
	}

	return nil
}
