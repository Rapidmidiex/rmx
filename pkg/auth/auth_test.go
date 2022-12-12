package auth

import (
	"net/http"
	"testing"
	"time"

	"github.com/hyphengolang/prelude/testing/is"
	"github.com/hyphengolang/prelude/types/email"
	"github.com/hyphengolang/prelude/types/password"
	"github.com/hyphengolang/prelude/types/suid"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/rog-golang-buddies/rmx/internal"
)

func TestToken(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	t.Run(`generate a token and sign`, func(t *testing.T) {
		_, private := ES256()

		u := internal.User{
			ID:       suid.NewUUID(),
			Username: "fizz_user",
			Email:    "fizz@mail.com",
			Password: password.Password("492045rf-vf").MustHash(),
		}

		o := TokenOption{
			Issuer:     "github.com/rog-golang-buddies/rmx",
			Subject:    suid.NewUUID().String(),
			Expiration: time.Hour * 10,
			Claims:     map[string]any{"email": u.Email},
		}

		_, err := Sign(private, &o)
		is.NoErr(err) // sign id token

		o.Subject = u.ID.String()
		o.Expiration = AccessTokenExpiry

		_, err = Sign(private, &o)
		is.NoErr(err) // access token

		o.Expiration = RefreshTokenExpiry
		_, err = Sign(private, &o)
		is.NoErr(err) // refresh token
	})
}

func TestMiddleware(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	t.Run("jwk parse request", func(t *testing.T) {
		public, private := ES256()

		e, cookieName := email.Email("foobar@gmail.com"), `__g`

		o := TokenOption{
			Issuer:     "github.com/rog-golang-buddies/rmx",
			Subject:    suid.NewUUID().String(),
			Expiration: time.Hour * 10,
			Claims:     map[string]any{"email": e.String()},
		}

		// rts
		rts, err := Sign(private, &o)
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

		//
		_, err = jwt.Parse([]byte(c.Value), jwt.WithKey(jwa.ES256, public), jwt.WithValidate(true))
		is.NoErr(err) // parsing jwk page not found
	})
}
