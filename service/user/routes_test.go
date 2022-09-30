package user

import (
	"net/http/httptest"
	"testing"
)

func TestRoutes(t *testing.T) {
	srv := DefaultService()

	s := httptest.NewServer(srv)
	t.Cleanup(func() { s.Close() })

	r, err := s.Client().Get(s.URL + "/api/v1/user/ping")
	if err != nil {
		t.Fatal(err)
	}

	if r.StatusCode != 204 {
		t.Fatalf("expected %d;got %d", 204, r.StatusCode)
	}
}
