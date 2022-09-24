package service

import (
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	// "github.com/rog-golang-buddies/rapidmidiex/internal/websocket"
)

func TestRoutes(t *testing.T) {
	srv := New(chi.NewMux())

	// srv.r.Get("/ws/echo", chain(srv.handleEcho(),srv.upgradeHTTP(1024,1024),srv.connectionPool(websocket.DefaultPool())))

	s := httptest.NewServer(srv)
	t.Cleanup(func() { s.Close() })

	r, err := s.Client().Get(s.URL + "/ping")
	if err != nil {
		t.Fatal(err)
	}

	if r.StatusCode != 204 {
		t.Fatalf("expected %d;got %d", 204, r.StatusCode)
	}
}
