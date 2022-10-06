package user

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthMe(t *testing.T) {
	srv := DefaultService()

	s := httptest.NewServer(srv)
	t.Cleanup(func() { s.Close() })

	r, err := s.Client().Get(s.URL + "/api/v1/account/me")
	if err != nil {
		t.Fatal(err)
	}

	if r.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected %d; got %d", http.StatusUnauthorized, r.StatusCode)
	}
}
