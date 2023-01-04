package rmx_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	jam "github.com/rapidmidiex/rmx/internal/jam"
	jamHTTP "github.com/rapidmidiex/rmx/internal/jam/http"
	"github.com/stretchr/testify/require"
)

func TestAcceptance(t *testing.T) {
	t.Run("As RMX client, I can create a Jam Session through the API", func(t *testing.T) {
		rmxSrv := jamHTTP.NewService(context.Background())
		// wsBase := rmxSrv.URL + "/ws"

		newJamResp := httptest.NewRecorder()

		// Client A creates a new Jam.
		jamName := "Jam On It!"
		newJamBody := fmt.Sprintf(`{"name":%q}`, jamName)
		newJamReq := newPostJamReq(t, strings.NewReader(newJamBody))
		rmxSrv.ServeHTTP(newJamResp, newJamReq)

		require.Equal(t, newJamResp.Result().StatusCode, http.StatusCreated)

		// Client should see the newly created Jam
		listJamsResp := httptest.NewRecorder()

		rmxSrv.ServeHTTP(listJamsResp, newGetJamsReq(t))
		require.Equal(t, listJamsResp.Result().StatusCode, http.StatusOK)

		listD := json.NewDecoder(listJamsResp.Body)
		var listJamsRespBody []jam.Jam
		listD.Decode(&listJamsRespBody)
		require.NotEmpty(t, listJamsRespBody[0])
		require.Equal(t, listJamsRespBody[0].Name, jamName)
		require.NotEmpty(t, listJamsRespBody[0].ID, "GET /jam Jams should have IDs")
	})

}

func newPostJamReq(t *testing.T, jamBody io.Reader) *http.Request {
	req, err := http.NewRequest(http.MethodPost, "/api/v1/jam", jamBody)
	require.NoError(t, err)
	return req
}

func newGetJamsReq(t *testing.T) *http.Request {
	req, err := http.NewRequest(http.MethodGet, "/api/v1/jam", nil)
	require.NoError(t, err)
	return req
}
