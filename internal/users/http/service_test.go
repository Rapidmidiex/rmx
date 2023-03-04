package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rapidmidiex/rmx/internal/users"
	"github.com/stretchr/testify/require"
)

var applicationJSON = "application/json"

func TestService(t *testing.T) {
	usersHTTP := New()
	srv := httptest.NewServer(usersHTTP)

	// CREATE A NEW USER WITH USERNAME
	{
		payload := `
		{
			"username": "test"
		}`
		resp, err := srv.Client().Post(srv.URL+"/", applicationJSON, strings.NewReader(payload))
		require.NoError(t, err, "creating a post request")
		require.Equal(t, http.StatusCreated, resp.StatusCode, "a 201 response code")

		defer resp.Body.Close()
		type response struct {
			User *users.User `json:"user"`
		}
		var data response
		err = json.NewDecoder(resp.Body).Decode(&data)
		require.NoError(t, err, "decoding the response body")

		require.NotEmpty(t, data.User.ID, "should have an ID")
		require.Equal(t, "test", data.User.Username)
	}
}
