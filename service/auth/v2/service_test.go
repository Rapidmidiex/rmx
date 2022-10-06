package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/rog-golang-buddies/rmx/internal/is"
	"github.com/rog-golang-buddies/rmx/store/db/v2/user"
)

const applicationJson = "application/json"

var s http.Handler

func init() {

	s = NewService(context.Background(), chi.NewMux(), user.MapRepo)
}

func TestService(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	srv := httptest.NewServer(s)
	t.Cleanup(func() { srv.Close() })

	t.Run("register a enw user", func(t *testing.T) {
		payload := `
		{
			"email":"fizz@gmail.com",
			"username":"fizz_user",
			"password":"fizz_$PW_10"
		}`

		res, _ := srv.Client().Post(srv.URL+"/api/v2/account/signup", applicationJson, strings.NewReader(payload))
		is.Equal(res.StatusCode, http.StatusCreated)
	})
}
