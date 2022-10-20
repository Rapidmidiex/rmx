package auth

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hyphengolang/prelude/types/email"
	"github.com/hyphengolang/prelude/types/password"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/rog-golang-buddies/rmx/internal"
	"github.com/rog-golang-buddies/rmx/internal/is"
	"github.com/rog-golang-buddies/rmx/internal/suid"
)

func TestToken(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	t.Run(`generate a token and sign`, func(t *testing.T) {
		kp := ES256()

		u := internal.User{
			ID:       suid.NewUUID(),
			Username: "fizz_user",
			Email:    "fizz@mail.com",
			Password: password.Password("492045rf-vf").MustHash(),
		}

		opt := TokenOption{
			Issuer:     "github.com/rog-golang-buddies/rmx",
			Subject:    suid.NewUUID().String(),
			Expiration: time.Hour * 10,
			Claims:     map[string]any{"email": u.Email.String()},
		}

		_, err := Sign(kp.Private(), &opt)
		is.NoErr(err) // sign id token

		opt.Subject = u.ID.String()
		opt.Expiration = AccessTokenExpiry
		_, err = Sign(kp.Private(), &opt)
		is.NoErr(err) // access token

		opt.Expiration = RefreshTokenExpiry
		_, err = Sign(kp.Private(), &opt)
		is.NoErr(err) // refresh token
	})
}

func TestMiddleware(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	t.Run("authenticate against Authorization header", func(t *testing.T) {
		kp := ES256()

		e := email.Email("foobar@gmail.com")

		opt := TokenOption{
			Issuer:     "github.com/rog-golang-buddies/rmx",
			Subject:    suid.NewUUID().String(),
			Expiration: time.Hour * 10,
			Claims:     map[string]any{"email": e.String()},
		}

		// ats
		ats, err := Sign(kp.Private(), &opt)
		is.NoErr(err) // signing access token

		h := ParseAuth(jwa.ES256, kp.Public())(http.NotFoundHandler())

		req, _ := http.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", string(ats)))

		res := httptest.NewRecorder()

		h.ServeHTTP(res, req)
		is.Equal(res.Result().StatusCode, http.StatusNotFound) // http page not found
	})

	t.Run("authenticate against Cookie header", func(t *testing.T) {
		kp := ES256()

		e, cookieName := email.Email("foobar@gmail.com"), `__myCookie`

		opt := TokenOption{
			Issuer:     "github.com/rog-golang-buddies/rmx",
			Subject:    suid.NewUUID().String(),
			Expiration: time.Hour * 10,
			Claims:     map[string]any{"email": e.String()},
		}

		// rts
		rts, err := Sign(kp.Private(), &opt)
		is.NoErr(err) // signing refresh token

		h := ParseAuth(jwa.ES256, kp.Public(), cookieName)(http.NotFoundHandler())

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

	t.Run("jwk parse request", func(t *testing.T) {
		kp := ES256()

		e, cookieName := email.Email("foobar@gmail.com"), `__g`

		opt := TokenOption{
			Issuer:     "github.com/rog-golang-buddies/rmx",
			Subject:    suid.NewUUID().String(),
			Expiration: time.Hour * 10,
			Claims:     map[string]any{"email": e.String()},
		}

		// rts
		rts, err := Sign(kp.Private(), &opt)
		is.NoErr(err) // signing refresh token

		c := &http.Cookie{
			Path:     "/",
			Name:     cookieName,
			Value:    string(rts),
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   24 * 7,
		}
		req, _ := http.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(c)

		_, err = jwt.Parse(
			[]byte(c.Value),
			jwt.WithKey(jwa.ES256, kp.Public()),
			jwt.WithValidate(true),
		)
		is.NoErr(err) // parsing jwk page not found
	})
}
