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
	jamDB "github.com/rapidmidiex/rmx/internal/jam/postgres"
	"github.com/rapidmidiex/rmx/internal/msg"
	"github.com/stretchr/testify/require"
)

type listJamsResponse struct {
	Rooms []room `json:"rooms"`
}

type room struct {
	jam.Jam
	PlayerCount int `json:"playerCount"`
}

func TestRESTAcceptance(t *testing.T) {
	t.Run("As RMX client, I can create a Jam Session through the API", func(t *testing.T) {
		rmxSrv := newTestService(t)

		// Client A creates a new Jam.
		jamName := "Jam On It!"
		newJamBody := strings.NewReader(fmt.Sprintf(`
		{
			"name":%q
		}`, jamName))

		newJamResp := testHandler(t, rmxSrv, http.MethodPost, "/", newJamBody, http.StatusCreated)
		defer newJamResp.Body.Close()

		var createdJam jam.Jam
		err := json.NewDecoder(newJamResp.Body).Decode(&createdJam)

		require.NoError(t, err)
		require.NotEmpty(t, createdJam.ID, "Jam should have an ID from the database")

		// Client should see the newly created Jam
		listJamsResp := testHandler(t, rmxSrv, http.MethodGet, "/", nil, http.StatusOK)
		defer listJamsResp.Body.Close()

		var listJamsRespBody listJamsResponse
		err = json.NewDecoder(listJamsResp.Body).Decode(&listJamsRespBody)
		require.NoError(t, err)

		rooms := listJamsRespBody.Rooms
		require.NotEmpty(t, rooms)
		require.NotEmpty(t, rooms[0])
		require.Equal(t, rooms[0].Name, jamName)
		require.NotEmpty(t, rooms[0].ID, "GET /jams Jams should have IDs")
	})

	t.Run("Service will set default 'name' and 'bpm'", func(t *testing.T) {
		srv := newTestService(t)

		resp := testHandler(t, srv, http.MethodPost, "/", strings.NewReader(`{}`), http.StatusCreated)
		defer resp.Body.Close()

		var created jam.Jam
		err := json.NewDecoder(resp.Body).Decode(&created)
		require.NoError(t, err)

		require.NotEmpty(t, created.BPM)
		require.NotEmpty(t, created.Name)
	})
}

func TestJamFlowAcceptance(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	// **** Create new Jam **** //
	jamName := "Jam On It!"
	newJamBody := strings.NewReader(fmt.Sprintf(`
	{
		"name":%q
	}`, jamName))

	newJamResp, err := srv.Client().Post(srv.URL+"/", "application/json", newJamBody)
	require.NoError(t, err)

	require.Equal(t, http.StatusCreated, newJamResp.StatusCode)

	defer newJamResp.Body.Close()

	var newJam jam.Jam
	err = json.NewDecoder(newJamResp.Body).Decode(&newJam)
	require.NoErrorf(t, err, "POST /jams should return the newly created Jam resource")

	// **** List Jams for selection **** //
	// Client would list the jams and select the one the want to join
	// or web client would auto-select the newly created Jam.
	// The request would be the same in either case.
	listJamResp, err := srv.Client().Get(srv.URL + "/")
	require.NoError(t, err)

	require.Equal(t, http.StatusOK, listJamResp.StatusCode, "GET /jams should return OK status")

	var jamsList listJamsResponse
	err = json.NewDecoder(listJamResp.Body).Decode(&jamsList)
	require.NoError(t, err)

	require.Len(t, jamsList.Rooms, 1, "should have one brand new Jam")

	// **** Use the Jam selection to join the Jam room **** //
	wsURL := fmt.Sprintf("%s/%s/ws", srv.URL, jamsList.Rooms[0].ID)

	// **** Client A joins Jam **** //
	// Intentionally external ws client (not rmx) because this client represent a client external to this system. (JS Frontend, TUI frontend)
	wsConnA, closeConnA := newConn(t, wsURL)
	defer closeConnA()

	// **** Client B joins Jam **** //
	var envelope msg.Envelope
	wsConnB, closeConnB := newConn(t, wsURL)
	defer closeConnB()

	// Check the player count
	playerCountResp, err := srv.Client().Get(srv.URL + "/")
	require.NoError(t, err)

	var gotRooms listJamsResponse
	err = json.NewDecoder(playerCountResp.Body).Decode(&gotRooms)
	require.NoError(t, err)

	require.Equal(t, 2, gotRooms.Rooms[0].PlayerCount, `"playerCount" field should be 2 since there are two active connections`)

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

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	ctx := context.Background()

	err := cleanDB(pgDB)
	require.NoError(t, err)

	h := jamHTTP.New(ctx, jamDB.New(pgDB))
	return httptest.NewServer(h)
}

func newTestService(t *testing.T) http.Handler {
	t.Helper()

	ctx := context.Background()

	err := cleanDB(pgDB)
	require.NoError(t, err)

	return jamHTTP.New(ctx, jamDB.New(pgDB))
}

func newConn(t *testing.T, url string) (conn *websocket.Conn, close func()) {
	t.Helper()

	var err error
	url = strings.Replace(url, "http", "ws", 1)

	conn, _, err = websocket.DefaultDialer.Dial(url, nil)
	require.NoErrorf(t, err, "client could not connect")

	close = func() {
		err := conn.WriteMessage(int(ws.OpClose), nil)
		require.NoError(t, err)
	}

	return conn, close
}

func testHandler(t *testing.T, h http.Handler, method string, url string, body io.Reader, code int) *http.Response {
	t.Helper()

	req, err := http.NewRequest(method, url, body)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	resp := rec.Result()

	require.Equal(t, resp.StatusCode, code)

	return resp
}
