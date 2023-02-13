package service_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/rapidmidiex/rmx/internal/jam"
	service "github.com/rapidmidiex/rmx/internal/jam/http"
)

func TestService(t *testing.T) {
	ctx := context.Background()

	service.NewService(ctx, newTestStore())

	srv := httptest.NewServer(http.NotFoundHandler())

	t.Cleanup(func() { srv.Close() })
}

type testStore struct{}

func newTestStore() *testStore {
	s := &testStore{}
	return s
}

func (s *testStore) CreateJam(context.Context, jam.Jam) (jam.Jam, error) {
	panic("implement me")
}

func (s *testStore) GetJams(context.Context) ([]jam.Jam, error) {
	panic("implement me")
}

func (s *testStore) GetJamByID(ctx context.Context, id uuid.UUID) (jam.Jam, error) {
	panic("implement me")
}
