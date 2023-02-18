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
	jamHTTP "github.com/rapidmidiex/rmx/internal/jam/http/v1"
	jamRepo "github.com/rapidmidiex/rmx/internal/jam/postgres"
	"github.com/rapidmidiex/rmx/internal/msg"
	"github.com/stretchr/testify/require"
)

type (
	listJamsResponse struct {
		Rooms []jam.Jam `json:"rooms"`
	}
)

func TestRESTAcceptance(t *testing.T) {
	t.Run("As RMX client, I can create a Jam Session through the API", func(t *testing.T) {
		err := cleanDB(pgDB)
		require.NoError(t, err)

		// FIXME I do not like this
		// store := db.Store{Q: testQueries}
		store := jamRepo.New(pgDB)

		rmxSrv := jamHTTP.NewService(context.Background(), store)
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
		require.NotEmpty(t, listJamsRespBody.Rooms[0].ID, "GET /jam Jams should have IDs")
	})

	t.Run("Service will set default 'name' and 'bpm'", func(t *testing.T) {
		err := cleanDB(pgDB)
		require.NoError(t, err)

		// FIXME I do not like this
		// store := db.Store{Q: testQueries}
		store := jamRepo.New(pgDB)

		rmxSrv := jamHTTP.NewService(context.Background(), store)

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
	store := jamRepo.New(pgDB)

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
	var jamsList listJamsResponse
	err = lD.Decode(&jamsList)
	require.NoError(t, err)

	require.Len(t, jamsList.Rooms, 1, "should have one brand new Jam")

	// **** Use the Jam selection to join the Jam room **** //
	// **** Client A joins Jam **** //
	roomID := jamsList.Rooms[0].ID
	jamWSurl := fmt.Sprintf("%s/jam/%s", wsBase, roomID)
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
	var envelope msg.Envelope
	var aConMsg msg.ConnectMsg
	err = wsConnA.ReadJSON(&envelope)
	require.NoError(t, err)

	err = json.Unmarshal(envelope.Payload, &aConMsg)
	require.NoError(t, err)
	userIDA := aConMsg.UserID

	// **** Client B joins Jam **** //
	wsConnB, _, err := websocket.DefaultDialer.Dial(jamWSurl, nil)
	require.NoErrorf(t, err, "client Bravo could not join Jam room: %q (%s)", newJam.Name, newJam.ID)
	defer func() {
		err := wsConnA.WriteMessage(int(ws.OpClose), nil)
		if err != nil {
			fmt.Println(err)
		}
	}()

	// Get user ID B from Connection Message
	var bConMsg msg.ConnectMsg
	err = wsConnB.ReadJSON(&envelope)
	require.NoError(t, err)
	require.Equal(t, msg.CONNECT, envelope.Typ, "should be a Connect message")
	err = json.Unmarshal(envelope.Payload, &bConMsg)
	require.NoError(t, err)
	userIDB := bConMsg.UserID
	require.NotEmpty(t, userIDB, "User B should have received a connect message containing their user ID")

	// Check the connection count
	getRoomsResp, err := http.Get(fmt.Sprintf("%s/jam", restBase))
	require.NoError(t, err)
	grd := json.NewDecoder(getRoomsResp.Body)

	type roomsResp struct {
		Rooms []struct {
			PlayerCount int `json:"playerCount"`
		} `json:"rooms"`
	}
	var gotRooms roomsResp
	err = grd.Decode(&gotRooms)
	require.NoError(t, err)

	require.Equal(t, 2, gotRooms.Rooms[0].PlayerCount, `"playerCount" field should be 2 since there are two active connections`)

	// Alpha sends a MIDI message
	// **** Client A broadcasts a MIDI message **** //
	yasiinSend := msg.MIDIMsg{
		State:  msg.NOTE_ON,
		Number: 60,
	}
	yasiinEnv := msg.Envelope{
		UserID: userIDA,
		Typ:    msg.MIDI,
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
