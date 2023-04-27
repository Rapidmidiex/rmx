package config

import (
	"reflect"
	"testing"
)

func TestConfig(t *testing.T) {
	// Write config to file
	i := &Config{
		Port: "8080",
		DB: DBConfig{
			Host:     "localhost",
			Port:     "3306",
			Name:     "rmx",
			User:     "rmx",
			Password: "password",
		},
		Auth: AuthConfig{
			Enable: true,
			Google: GoogleConfig{
				ClientID:     "client_id",
				ClientSecret: "client_secret",
			},
			CookieKey: "cookie_key",
		},
		Dev: true,
	}

	if err := i.WriteToFile(); err != nil {
		t.Fatal(err)
	}

	// Read config from file
	o, err := ScanConfigFile()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(i, o) {
		t.Fatalf("expected:\n%+v\ngot:\n%+v", i, o)
	}
}
