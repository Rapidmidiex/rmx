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

	"github.com/gobwas/ws"
	"github.com/gorilla/websocket"
	jam "github.com/rapidmidiex/rmx/internal/jam"
	jamHTTP "github.com/rapidmidiex/rmx/internal/jam/http"
	jamDB "github.com/rapidmidiex/rmx/internal/jam/store"
	"github.com/rapidmidiex/rmx/internal/msg"
	"github.com/stretchr/testify/require"
)

type (
	listJamsResponse struct {
		Rooms []room `json:"rooms"`
	}

	room struct {
		jam.Jam
		PlayerCount int `json:"playerCount"`
	}
)

func TestRESTAcceptance(t *testing.T) {
	t.Run("As RMX client, I can create a Jam Session through the API", func(t *testing.T) {
		err := cleanDB(pgDB)
		require.NoError(t, err)

		store := jamDB.New(pgDB)

		rmxSrv := jamHTTP.New(context.Background(), store)
		// wsBase := rmxSrv.URL + "/ws"

		newJamResp := httptest.NewRecorder()

		// Client A creates a new Jam.
		jamName := "Jam On It!"
		newJamBody := fmt.Sprintf(`{"name":%q}`, jamName)
		newJamReq := newPostJamReq(t, strings.NewReader(newJamBody))
		rmxSrv.ServeHTTP(newJamResp, newJamReq)

		require.Equal(t, newJamResp.Result().StatusCode, http.StatusCreated)
		var createdJam jam.Jam
		d := json.NewDecoder(newJamResp.Body)
		err = d.Decode(&createdJam)
		require.NoError(t, err)

		require.NotEmpty(t, createdJam.ID, "Jam should have an ID from the database")
		// Client should see the newly created Jam
		listJamsResp := httptest.NewRecorder()

		rmxSrv.ServeHTTP(listJamsResp, newGetJamsReq(t))
		require.Equal(t, listJamsResp.Result().StatusCode, http.StatusOK)

		listD := json.NewDecoder(listJamsResp.Body)
		var listJamsRespBody listJamsResponse
		err = listD.Decode(&listJamsRespBody)
		require.NoError(t, err)

		require.NotEmpty(t, listJamsRespBody.Rooms)
		require.NotEmpty(t, listJamsRespBody.Rooms[0])
		require.Equal(t, listJamsRespBody.Rooms[0].Name, jamName)
		require.NotEmpty(t, listJamsRespBody.Rooms[0].ID, "GET /jams Jams should have IDs")
	})

	t.Run("Service will set default 'name' and 'bpm'", func(t *testing.T) {
		err := cleanDB(pgDB)
		require.NoError(t, err)

		// FIXME I do not like this
		// store := db.Store{Q: testQueries}
		store := jamDB.New(pgDB)

		rmxSrv := jamHTTP.New(context.Background(), store)

		newJamResp := httptest.NewRecorder()
		// Send empty JSON body
		newJamReq := newPostJamReq(t, strings.NewReader(`{}`))
		rmxSrv.ServeHTTP(newJamResp, newJamReq)

		require.Equal(t, newJamResp.Result().StatusCode, http.StatusCreated)
		var createdJam jam.Jam
		d := json.NewDecoder(newJamResp.Body)
		err = d.Decode(&createdJam)
		require.NoError(t, err)

		require.NotEmpty(t, createdJam.BPM)
		require.NotEmpty(t, createdJam.Name)
	})
}

func TestJamFlowAcceptance(t *testing.T) {
	err := cleanDB(pgDB)
	require.NoError(t, err)

	// FIXME I do not like this
	// store := db.Store{Q: testQueries}
	store := jamDB.New(pgDB)

	jamSvc := jamHTTP.New(context.Background(), store)
	rmxSrv := httptest.NewServer(jamSvc)
	defer rmxSrv.Close()

	restBase := rmxSrv.URL
	wsBase := strings.Replace(restBase, "http", "ws", 1)

	// **** Create new Jam **** //
	jamName := "Jam On It!"
	newJamBody := fmt.Sprintf(`{"name":%q}`, jamName)
	newJamResp, err := http.Post(restBase, "application/json", strings.NewReader(newJamBody))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, newJamResp.StatusCode)

	var newJam jam.Jam
	jD := json.NewDecoder(newJamResp.Body)
	err = jD.Decode(&newJam)
	require.NoErrorf(t, err, "POST /jams should return the newly created Jam resource")

	// **** List Jams for selection **** //
	// Client would list the jams and select the one the want to join
	// or web client would auto-select the newly created Jam.
	// The request would be the same in either case.
	listJamResp, err := http.Get(restBase)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, listJamResp.StatusCode, "GET /jams should return OK status")

	lD := json.NewDecoder(listJamResp.Body)
	var jamsList listJamsResponse
	err = lD.Decode(&jamsList)
	require.NoError(t, err)

	require.Len(t, jamsList.Rooms, 1, "should have one brand new Jam")

	// **** Use the Jam selection to join the Jam room **** //
	// **** Client A joins Jam **** //
	roomID := jamsList.Rooms[0].ID
	jamWSurl := fmt.Sprintf("%s/%s/ws", wsBase, roomID)
	// Intentionally external ws client (not rmx) because this client represent a client external to this system. (JS Frontend, TUI frontend)
	wsConnA, _, err := websocket.DefaultDialer.Dial(jamWSurl, nil)
	// TODO: Fails. We should be able to join a Jam with the Jam ID. The service should figure out the rest
	require.NoErrorf(t, err, "client Alpha could not join Jam room: %q (%s)", newJam.Name, newJam.ID)
	defer func() {
		err := wsConnA.WriteMessage(int(ws.OpClose), nil)
		if err != nil {
			fmt.Println(err)
		}
	}()

	// Get user ID from Connection Message
	// var envelope msg.Envelope
	// var aConMsg msg.ConnectMsg
	// err = wsConnA.ReadJSON(&envelope)
	// require.NoError(t, err)

	// err = json.Unmarshal(envelope.Payload, &aConMsg)
	// require.NoError(t, err)
	// userIDA := aConMsg.UserID

	// **** Client B joins Jam **** //
	var envelope msg.Envelope
	wsConnB, _, err := websocket.DefaultDialer.Dial(jamWSurl, nil)
	require.NoErrorf(t, err, "client Bravo could not join Jam room: %q (%s)", newJam.Name, newJam.ID)
	defer func() {
		err := wsConnB.WriteMessage(int(ws.OpClose), nil)
		if err != nil {
			fmt.Println(err)
		}
	}()

	// Check the player count
	playerCountResp, err := http.Get(restBase)
	require.NoError(t, err)

	d := json.NewDecoder(playerCountResp.Body)
	var gotRooms listJamsResponse
	err = d.Decode(&gotRooms)
	require.NoError(t, err)

	require.Equal(t, 2, gotRooms.Rooms[0].PlayerCount, `"playerCount" field should be 2 since there are two active connections`)

	// Get user ID B from Connection Message
	// var bConMsg msg.ConnectMsg
	// err = wsConnB.ReadJSON(&envelope)
	// require.NoError(t, err)
	// require.Equal(t, msg.CONNECT, envelope.Typ, "should be a Connect message")
	// err = json.Unmarshal(envelope.Payload, &bConMsg)
	// require.NoError(t, err)
	// userIDB := bConMsg.UserID
	// require.NotEmpty(t, userIDB, "User B should have received a connect message containing their user ID")

	// Alpha sends a MIDI message
	// **** Client A broadcasts a MIDI message **** //
	yasiinSend := msg.MIDIMsg{
		State:  msg.NOTE_ON,
		Number: 60,
	}
	yasiinEnv := msg.Envelope{
		// UserID: userIDA,
		Typ: msg.MIDI,
	}
	err = yasiinEnv.SetPayload(yasiinSend)
	require.NoError(t, err)

	err = wsConnA.WriteJSON(yasiinEnv)
	require.NoErrorf(t, err, "Client A, %q, could not write MIDI note to connection", yasiinEnv.UserID)

	// **** Client B receives the MIDI message **** //
	// Talib should be able to hear MIDI note
	var talibRecv msg.MIDIMsg
	err = wsConnB.ReadJSON(&envelope)
	require.NoError(t, err, "Client B could not read MIDI note from connection")
	require.Equal(t, msg.MIDI, envelope.Typ, "should be a MIDI message")
	err = envelope.Unwrap(&talibRecv)
	require.NoError(t, err, "could not unwrap client B's message")
	require.Equal(t, yasiinSend, talibRecv, "Talib received MIDI message does not match what Yasiin sent")
}

// newPostJamReq creates a POST /jams request to REST API to create a new Jam.
func newPostJamReq(t *testing.T, jamBody io.Reader) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, "/", jamBody)
	require.NoError(t, err)
	return req
}

// newGetJamsReq creates a GET /jams request to REST API to list available Jams.
func newGetJamsReq(t *testing.T) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)
	return req
}
