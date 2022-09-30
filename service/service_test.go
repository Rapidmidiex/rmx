package service

import (
	"net/http/httptest"
	"testing"
)

func TestIntegration(t *testing.T) {
	srv := Default()

	s := httptest.NewServer(srv)
	t.Cleanup(func() { s.Close() })

	// jam service
	r, err := s.Client().Get(s.URL + "/api/v1/jam/ping")
	if err != nil {
		t.Fatal(err)
	}

	if r.StatusCode != 204 {
		t.Fatalf("expected %d;got %d", 204, r.StatusCode)
	}

	// user service
	r, err = s.Client().Get(s.URL + "/api/v1/user/ping")
	if err != nil {
		t.Fatal(err)
	}

	if r.StatusCode != 204 {
		t.Fatalf("expected %d;got %d", 204, r.StatusCode)
	}
}
