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
	jam "github.com/rapidmidiex/rmx/internal/jam"
	jamHTTP "github.com/rapidmidiex/rmx/internal/jam/http"
	"github.com/stretchr/testify/require"
)

func TestRESTAcceptance(t *testing.T) {
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

func TestJamFlowAcceptance(t *testing.T) {
	rmxSrv := httptest.NewServer(jamHTTP.NewService(context.Background()))
	defer rmxSrv.Close()

	restBase := rmxSrv.URL + "/ap1/v1"
	wsBase := "ws" + strings.TrimPrefix(rmxSrv.URL, "http") + "/ws"
	// *********
	// Create new Jam
	restClientA := http.DefaultClient
	jamName := "Jam On It!"
	newJamBody := fmt.Sprintf(`{"name":%q}`, jamName)
	newJamResp, err := restClientA.Post(restBase+"/jam", "application/json", strings.NewReader(newJamBody))
	require.NoError(t, err)

	var newJam jam.Jam
	jD := json.NewDecoder(newJamResp.Body)
	err = jD.Decode(&newJam)
	require.NoErrorf(t, err, "POST /jam should return the newly created Jam resource")

	// *********
	// Client would list the jams and select the one the want to join
	// or web client would auto-select the newly created Jam.
	// The request would be the same in either case.
	listJamResp, err := restClientA.Get(restBase + "/jam")
	require.NoError(t, err)

	lD := json.NewDecoder(listJamResp.Body)
	var jamsList []jam.Jam
	err = lD.Decode(&jamsList)
	require.NoError(t, err)
	require.Len(t, jamsList, 1, "should have one brand new Jam")

	// *********
	// Use the Jam selection to join the Jam room
	jamWSurl := fmt.Sprintf("%s/jam/%s", wsBase, jamsList[0].ID)
	// Intentionally external ws client (not rmx) because this client represent a client external to this system. (JS Frontend, TUI frontend)
	wsConnA, _, err := websocket.DefaultDialer.Dial(jamWSurl, nil)
	require.NoErrorf(t, err, "client Alpha could not join Jam room: %q (%s)", newJam.Name, newJam.ID)
	defer wsConnA.Close()

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

	yasiinSend := midiMsg{
		Typ:      "NOTE_ON",
		Note:     60,
		Velocity: 127,
		ClientID: "Yasiin Bey",
	}
	err = wsConnA.WriteJSON(yasiinSend)
	require.NoErrorf(t, err, "Client A, %q, could not write MIDI note to connection", yasiinSend.ClientID)

	// Talib should be albe to hear MIDI note
	var talibRecv midiMsg
	err = wsConnB.ReadJSON(talibRecv)
	require.NoError(t, err, "Client B could not read MIDI note from connection")
	require.Equal(t, yasiinSend, talibRecv, "Talib received MIDI message does not match what Yasiin sent")
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
