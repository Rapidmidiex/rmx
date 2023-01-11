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

	"github.com/gorilla/websocket"
	"github.com/rapidmidiex/rmx/internal/db"
	jam "github.com/rapidmidiex/rmx/internal/jam"
	jamHTTP "github.com/rapidmidiex/rmx/internal/jam/http"
	"github.com/stretchr/testify/require"
)

func TestRESTAcceptance(t *testing.T) {
	t.Run("As RMX client, I can create a Jam Session through the API", func(t *testing.T) {
		err := cleanDB(pgdb)
		require.NoError(t, err)

		store := db.Store{Q: testQueries}
		rmxSrv := jamHTTP.NewService(context.Background(), store)
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
		err = listD.Decode(&listJamsRespBody)
		require.NoError(t, err)

		require.NotEmpty(t, listJamsRespBody)
		require.NotEmpty(t, listJamsRespBody[0])
		require.Equal(t, listJamsRespBody[0].Name, jamName)
		require.NotEmpty(t, listJamsRespBody[0].ID, "GET /jam Jams should have IDs")
	})
}

func TestJamFlowAcceptance(t *testing.T) {
	err := cleanDB(pgdb)
	require.NoError(t, err)

	store := db.Store{Q: testQueries}
	jamSvc := jamHTTP.NewService(
		context.Background(),
		store,
	)
	rmxSrv := httptest.NewServer(jamSvc)
	defer rmxSrv.Close()

	restBase := rmxSrv.URL + "/api/v1"
	wsBase := "ws" + strings.TrimPrefix(rmxSrv.URL, "http") + "/ws"

	// **** Create new Jam **** //
	jamName := "Jam On It!"
	newJamBody := fmt.Sprintf(`{"name":%q}`, jamName)
	newJamResp, err := http.Post(restBase+"/jam", "application/json", strings.NewReader(newJamBody))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, newJamResp.StatusCode)

	var newJam jam.Jam
	jD := json.NewDecoder(newJamResp.Body)
	err = jD.Decode(&newJam)
	require.NoErrorf(t, err, "POST /jam should return the newly created Jam resource")

	// **** List Jams for selection **** //
	// Client would list the jams and select the one the want to join
	// or web client would auto-select the newly created Jam.
	// The request would be the same in either case.
	listJamResp, err := http.Get(restBase + "/jam/")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, listJamResp.StatusCode, "GET /jam should return OK status")

	lD := json.NewDecoder(listJamResp.Body)
	var jamsList []jam.Jam
	err = lD.Decode(&jamsList)
	require.NoError(t, err)

	require.Len(t, jamsList, 1, "should have one brand new Jam")

	// **** Use the Jam selection to join the Jam room **** //
	// **** Client A joins Jam **** //
	jamWSurl := fmt.Sprintf("%s/jam/%s", wsBase, jamsList[0].ID)
	// Intentionally external ws client (not rmx) because this client represent a client external to this system. (JS Frontend, TUI frontend)
	wsConnA, _, err := websocket.DefaultDialer.Dial(jamWSurl, nil)
	// TODO: Fails. We should be able to join a Jam with the Jam ID. The service should figure out the rest
	require.NoErrorf(t, err, "client Alpha could not join Jam room: %q (%s)", newJam.Name, newJam.ID)
	defer wsConnA.Close()

	// **** Client B joins Jam **** //
	wsConnB, _, err := websocket.DefaultDialer.Dial(jamWSurl, nil)
	require.NoErrorf(t, err, "client Bravo could not join Jam room: %q (%s)", newJam.Name, newJam.ID)
	defer wsConnB.Close()

	// Alpha sends a MIDI message
	type midiMsg struct {
		Typ string `json:"type"` // NOTE_ON | NOTE_OFF
		// MIDI Note # in "C3 Convention", C3 = 60. Available values: (0-127)
		Note int `json:"note"`
		// MIDI Velocity (0-127)
		Velocity int `json:"velocity"`
		// RMX client identifier
		ClientID string `json:"clientID"`
	}

	// **** Client A broadcasts a MIDI message **** //
	yasiinSend := midiMsg{
		Typ:      "NOTE_ON",
		Note:     60,
		Velocity: 127,
		ClientID: "Yasiin Bey",
	}
	err = wsConnA.WriteJSON(yasiinSend)
	require.NoErrorf(t, err, "Client A, %q, could not write MIDI note to connection", yasiinSend.ClientID)

	// **** Client B receives the MIDI message **** //
	// Talib should be able to hear MIDI note
	var talibRecv midiMsg
	err = wsConnB.ReadJSON(&talibRecv)
	require.NoError(t, err, "Client B could not read MIDI note from connection")
	require.Equal(t, yasiinSend, talibRecv, "Talib received MIDI message does not match what Yasiin sent")
}

// newPostJamReq creates a POST /jam request to REST API to create a new Jam.
func newPostJamReq(t *testing.T, jamBody io.Reader) *http.Request {
	req, err := http.NewRequest(http.MethodPost, "/api/v1/jam", jamBody)
	require.NoError(t, err)
	return req
}

// newGetJamsReq creates a GET /jam request to REST API to list available Jams.
func newGetJamsReq(t *testing.T) *http.Request {
	req, err := http.NewRequest(http.MethodGet, "/api/v1/jam", nil)
	require.NoError(t, err)
	return req
}
