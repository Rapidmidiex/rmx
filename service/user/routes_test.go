package user

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLogin(t *testing.T) {
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
		t.Fatalf("expected %d; got %d", http.StatusOK, r.StatusCode)
	}

	defer r.Body.Close()

	type response struct {
		IDToken     string `json:"idToken"`
		AccessToken string `json:"accessToken"`
		// PublicKey   string `json:"publicKey"`
	}

	var tokens response
	if err := json.NewDecoder(r.Body).Decode(&tokens); err != nil {
		t.Fatal(err)
	}

	// get my user info
	req, _ := http.NewRequest(http.MethodGet, s.URL+"/api/v1/user/me", nil)
	req.Header.Set(`Authorization`, fmt.Sprintf(`Bearer %s`, tokens.IDToken))

	r, err = s.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}

	if r.StatusCode != http.StatusOK {
		t.Fatalf("expected %d; got %d", http.StatusOK, r.StatusCode)
	}

	var str string
	if err := json.NewDecoder(r.Body).Decode(&str); err != nil {
		t.Fatal(err)
	}

	t.Log(str)
}

func TestAuthMe(t *testing.T) {
	srv := DefaultService()

	s := httptest.NewServer(srv)
	t.Cleanup(func() { s.Close() })

	r, err := s.Client().Get(s.URL + "/api/v1/user/me")
	if err != nil {
		t.Fatal(err)
	}

	if r.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected %d; got %d", http.StatusUnauthorized, r.StatusCode)
	}

}
