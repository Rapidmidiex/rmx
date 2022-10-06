package auth

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/rog-golang-buddies/rmx/internal"
	"github.com/rog-golang-buddies/rmx/internal/fp"
	"github.com/rog-golang-buddies/rmx/internal/is"
	"github.com/rog-golang-buddies/rmx/internal/suid"
)

var rsaPrivateKey = `-----BEGIN PRIVATE KEY-----
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

func TestToken(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	t.Run(`generate a token and sign`, func(t *testing.T) {
		key := NewPairES256()

		u := internal.User{
			ID:       suid.NewUUID(),
			Username: "fizz_user",
			Email:    "fizz@mail.com",
			Password: internal.Password("492045rf-vf").MustHash(),
		}

		opt := TokenOption{
			Issuer:     "github.com/rog-golang-buddies/rmx",
			Subject:    suid.NewUUID().String(),
			Expiration: time.Hour * 10,
			Claims:     []fp.Tuple{{"email", u.Email.String()}},
			Algo:       jwa.ES256,
		}

		_, err := SignToken(key.Private(), &opt)
		is.NoErr(err) // sign id token

		opt.Subject = u.ID.String()
		opt.Expiration = AccessTokenExpiry
		_, err = SignToken(key.Private(), &opt)
		is.NoErr(err) // access token

		opt.Expiration = RefreshTokenExpiry
		_, err = SignToken(key.Private(), &opt)
		is.NoErr(err) // refresh token
	})
}

func TestMiddleware(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	t.Run("authenticate against Authorization header", func(t *testing.T) {
		key := NewPairES256()

		e := internal.Email("foobar@gmail.com")

		opt := TokenOption{
			Issuer:     "github.com/rog-golang-buddies/rmx",
			Subject:    suid.NewUUID().String(),
			Expiration: time.Hour * 10,
			Claims:     []fp.Tuple{{"email", e.String()}},
			Algo:       jwa.ES256,
		}

		// ats
		ats, err := SignToken(key.Private(), &opt)
		is.NoErr(err) // signing access token

		h := Authenticate(opt.Algo, key.Public())(http.NotFoundHandler())

		req, _ := http.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", string(ats)))

		res := httptest.NewRecorder()

		h.ServeHTTP(res, req)
		is.Equal(res.Result().StatusCode, http.StatusNotFound) // return not found
	})

	t.Run("authenticate against Cookie header", func(t *testing.T) {
		key := NewPairES256()

		e := internal.Email("foobar@gmail.com")

		opt := TokenOption{
			Issuer:     "github.com/rog-golang-buddies/rmx",
			Subject:    suid.NewUUID().String(),
			Expiration: time.Hour * 10,
			Claims:     []fp.Tuple{{"email", e.String()}},
			Algo:       jwa.ES256,
		}

		// rts
		rts, err := SignToken(key.Private(), &opt)
		is.NoErr(err) // signing refresh token

		cookieName := `__myCookie`

		h := AuthenticateRefresh(opt.Algo, key.Public(), cookieName)(http.NotFoundHandler())

		req, _ := http.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{
			Path:     "/",
			Name:     cookieName,
			Value:    string(rts),
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   24 * 7,
		})

		res := httptest.NewRecorder()

		h.ServeHTTP(res, req)
		is.Equal(res.Result().StatusCode, http.StatusNotFound) // http page not found
	})

}
