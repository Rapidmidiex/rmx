package auth

import (
	"encoding/json"
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

	r, err := s.Client().
		Post(s.URL+"/api/v1/auth/register", "application/json", strings.NewReader(payload))
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

	r, err = s.Client().
		Post(s.URL+"/api/v1/auth/login", "application/json", strings.NewReader(payload))
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
	}

	var tokens response
	if err := json.NewDecoder(r.Body).Decode(&tokens); err != nil {
		t.Fatal(err)
	}

	/*
		// get my user info
		req, _ := http.NewRequest(http.MethodGet, s.URL+"/api/v1/account/me", nil)
		req.Header.Set(`Authorization`, fmt.Sprintf(`Bearer %s`, tokens.IDToken))
		r, err = s.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}

		if r.StatusCode != http.StatusOK {
			t.Fatalf("expected %d; got %d", http.StatusOK, r.StatusCode)
		}

		var str any
		if err := json.NewDecoder(r.Body).Decode(&str); err != nil {
			t.Fatal(err)
		}
	*/
}