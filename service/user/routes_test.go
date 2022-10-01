package user

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRoutes(t *testing.T) {
	srv := DefaultService()

	s := httptest.NewServer(srv)
	t.Cleanup(func() { s.Close() })

	// TODO add tests when -
	// `password`, `email` or `username` is not present
	payload := `
{
	"username":"Test User", 
	"password":"difficultPassword",
	"email":"user@gmail.com"
}`

	r, err := s.Client().Post(s.URL+"/api/v1/user", "application/json", strings.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}

	if r.StatusCode != http.StatusCreated {
		t.Fatalf("expected %d; got %d", http.StatusCreated, r.StatusCode)
	}

	payload = `
{
	"email":"user@gmail.com",
	"password":"difficultPassword"
}`

	r, err = s.Client().Post(s.URL+"/api/v1/auth/login", "application/json", strings.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}

	if r.StatusCode != http.StatusOK {
		t.Fatalf("expected %d; got %d", http.StatusCreated, r.StatusCode)
	}
}
