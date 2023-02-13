package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/rapidmidiex/rmx/internal/jam"
	service "github.com/rapidmidiex/rmx/internal/jam/http/v2"
	"github.com/stretchr/testify/require"
)

var applicationJSON = "application/json"

func TestService(t *testing.T) {
	ctx := context.Background()

	h := service.New(ctx, newTestStore())

	srv := httptest.NewServer(h)

	t.Cleanup(func() { srv.Close() })

	var roomID uuid.UUID

	t.Run("POST /api/v1/jam", func(t *testing.T) {
		payload := `
		{
			"name": "room-1",
			"capacity": 2,
			"bpm": 120
		}`
		log.Println(srv.URL + "/")
		resp, err := srv.Client().Post(srv.URL+"/api/v1/jam", applicationJSON, strings.NewReader(payload))
		require.NoError(t, err, "should not error")
		require.Equal(t, http.StatusCreated, resp.StatusCode, "should return 201")

		// get uuid from body
		defer resp.Body.Close()

		var jam jam.Jam
		err = json.NewDecoder(resp.Body).Decode(&jam)
		require.NoError(t, err, "should not error")

		require.NotEmpty(t, jam.ID, "should have an ID")

		roomID = jam.ID
	})

	t.Run("GET /api/v1/jam/{uuid}", func(t *testing.T) {

		log.Println(srv.URL + "/api/v1/jam/" + roomID.String())

		resp, err := srv.Client().Get(srv.URL + "/api/v1/jam/" + roomID.String())
		require.NoError(t, err, "should not error")
		require.Equal(t, http.StatusOK, resp.StatusCode, "should return 200")

		defer resp.Body.Close()

		var jam jam.Jam
		err = json.NewDecoder(resp.Body).Decode(&jam)
		require.NoError(t, err, "should not error")

		require.Equal(t, roomID, jam.ID, "should have the same ID")
	})

	t.Run("GET /ws/v1/jam/{uuid}", func(t *testing.T) {
		// create a new websocket connection
	})
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
