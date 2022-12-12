package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/hyphengolang/prelude/testing/is"
	"github.com/rog-golang-buddies/rmx/pkg/repotest"
	"github.com/rog-golang-buddies/rmx/store/auth"
)

const applicationJson = "application/json"

var s http.Handler

func init() {
	ctx, mux := context.Background(), chi.NewMux()

	s = NewService(ctx, mux, repotest.NewUserRepo(), auth.DefaultTokenClient)
}

func TestService(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	srv := httptest.NewServer(s)
	t.Cleanup(func() { srv.Close() })

	t.Run("register a new user", func(t *testing.T) {
		payload := `
		{
			"email":"fizz@gmail.com",
			"username":"fizz_user",
			"password":"fizz_$PW_10"
		}`

		res, _ := srv.Client().
			Post(srv.URL+"/api/v1/auth/sign-up", applicationJson, strings.NewReader(payload))
		is.Equal(res.StatusCode, http.StatusCreated)
	})

	t.Run("sign-in, access auth endpoint then sign-out", func(t *testing.T) {
		payload := `
		{
			"email":"fizz@gmail.com",
			"password":"fizz_$PW_10"
		}`

		res, _ := srv.Client().
			Post(srv.URL+"/api/v1/auth/sign-in", applicationJson, strings.NewReader(payload))
		is.Equal(res.StatusCode, http.StatusOK)

		type body struct {
			IDToken     string `json:"idToken"`
			AccessToken string `json:"accessToken"`
		}

		var b body
		err := json.NewDecoder(res.Body).Decode(&b)
		res.Body.Close()
		is.NoErr(err) // parsing json

		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/account/me", nil)
		req.Header.Set(`Authorization`, fmt.Sprintf(`Bearer %s`, b.AccessToken))
		res, _ = srv.Client().Do(req)
		is.Equal(res.StatusCode, http.StatusOK) // authorized endpoint

		req, _ = http.NewRequest(http.MethodDelete, srv.URL+"/api/v1/auth/sign-out", nil)
		req.Header.Set(`Authorization`, fmt.Sprintf(`Bearer %s`, b.AccessToken))
		res, _ = srv.Client().Do(req)
		is.Equal(res.StatusCode, http.StatusOK) // delete cookie
	})

	t.Run("refresh token", func(t *testing.T) {
		payload := `
		{
			"email":"fizz@gmail.com",
			"password":"fizz_$PW_10"
		}`

		res, _ := srv.Client().
			Post(srv.URL+"/api/v1/auth/sign-in", applicationJson, strings.NewReader(payload))
		is.Equal(res.StatusCode, http.StatusOK) // add refresh token

		// get the refresh token from the response's `Set-Cookie` header
		c := &http.Cookie{}
		for _, k := range res.Cookies() {
			t.Log(k.Value)
			if k.Name == cookieName {
				c = k
			}
		}

		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/auth/refresh", nil)
		req.AddCookie(c)

		res, _ = srv.Client().Do(req)
		is.Equal(res.StatusCode, http.StatusOK) // refresh token
	})
}
