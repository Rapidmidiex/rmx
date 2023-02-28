package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gobwas/ws"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rapidmidiex/rmx/internal/jam"
	service "github.com/rapidmidiex/rmx/internal/jam/http"
	"github.com/rapidmidiex/rmx/internal/msg"
	"github.com/stretchr/testify/require"
)

var applicationJSON = "application/json"

func TestService(t *testing.T) {
	ctx := context.Background()

	h := service.New(ctx, newTestStore())

	srv := httptest.NewServer(h)

	t.Cleanup(func() { srv.Close() })

	var roomID uuid.UUID

	/* "POST /v0/jams" */
	{
		payload := `
		{
			"name": "room-1",
			"capacity": 2,
			"bpm": 120
		}`
		resp, err := srv.Client().Post(srv.URL+"/v0/jams", applicationJSON, strings.NewReader(payload))
		require.NoError(t, err, "should not error")
		require.Equal(t, http.StatusCreated, resp.StatusCode, "should return 201")

		// get uuid from body
		defer resp.Body.Close()

		var jam jam.Jam
		err = json.NewDecoder(resp.Body).Decode(&jam)
		require.NoError(t, err, "should not error")

		require.NotEmpty(t, jam.ID, "should have an ID")

		roomID = jam.ID
	}

	/* GET /v0/jams/{uuid} */
	{
		log.Println(srv.URL + "/v0/jams/" + roomID.String())

		resp, err := srv.Client().Get(srv.URL + "/v0/jams/" + roomID.String())
		require.NoError(t, err, "should not error")
		require.Equal(t, http.StatusOK, resp.StatusCode, "should return 200")

		defer resp.Body.Close()

		var jam jam.Jam
		err = json.NewDecoder(resp.Body).Decode(&jam)
		require.NoError(t, err, "should not error")

		require.Equal(t, roomID, jam.ID, "should have the same ID")
	}

	/* "connect to websocket pool" */
	{
		// create a new websocket connection
		// **** Use the Jam selection to join the Jam room **** //
		wsBase := "ws" + strings.TrimPrefix(srv.URL, "http") + "/v0"
		jamWSurl := fmt.Sprintf("%s/jams/%s/ws", wsBase, roomID)

		// **** Client A joins Jam **** //
		wsConnA, _, err := websocket.DefaultDialer.Dial(jamWSurl, nil)
		require.NoErrorf(t, err, "client Alpha could not join Jam room")
		defer func() {
			err := wsConnA.WriteMessage(int(ws.OpClose), nil)
			if err != nil {
				fmt.Println(err)
			}
		}()

		/* TODO: this needs to pass before this PR can even be allowed
		need to resolve this by sending message on connection */
		{
			// Get user ID from Connection Message
			// var envelope msg.Envelope
			// var aConMsg msg.ConnectMsg
			// err = wsConnA.ReadJSON(&envelope)
			// require.NoError(t, err)

			// err = json.Unmarshal(envelope.Payload, &aConMsg)
			// require.NoError(t, err)
			// userIDA := aConMsg.UserID
		}

		// **** Client B joins Jam **** //
		var envelope msg.Envelope
		wsConnB, _, err := websocket.DefaultDialer.Dial(jamWSurl, nil)
		require.NoErrorf(t, err, "client Bravo could not join Jam room")
		defer func() {
			err := wsConnA.WriteMessage(int(ws.OpClose), nil)
			if err != nil {
				fmt.Println(err)
			}
		}()

		/* TODO: this needs to pass before this PR can even be allowed
		need to resolve this by sending message on connection */
		{
			// Get user ID B from Connection Message
			// var bConMsg msg.ConnectMsg
			// err = wsConnB.ReadJSON(&envelope)
			// require.NoError(t, err)
			// require.Equal(t, msg.CONNECT, envelope.Typ, "should be a Connect message")
			// err = json.Unmarshal(envelope.Payload, &bConMsg)
			// require.NoError(t, err)
			// userIDB := bConMsg.UserID
			// require.NotEmpty(t, userIDB, "User B should have received a connect message containing their user ID")
		}

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
}

type testStore struct {
	mu sync.Mutex
	m  map[uuid.UUID]jam.Jam
}

func newTestStore() *testStore {
	s := &testStore{
		m: make(map[uuid.UUID]jam.Jam),
	}
	return s
}

func (s *testStore) CreateJam(ctx context.Context, j jam.Jam) (jam.Jam, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	created := jam.Jam{
		ID:       uuid.New(),
		Name:     j.Name,
		Capacity: j.Capacity,
		BPM:      j.BPM,
	}

	s.m[created.ID] = created
	return created, nil
}

func (s *testStore) GetJams(context.Context) ([]jam.Jam, error) {
	panic("implement me")
}

func (s *testStore) GetJamByID(ctx context.Context, id uuid.UUID) (jam.Jam, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.m[id]
	if !ok {
		return jam.Jam{}, errors.New("jam not found")
	}

	return j, nil
}
