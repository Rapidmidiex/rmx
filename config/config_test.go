package config

import (
	"reflect"
	"testing"
)

func TestConfig(t *testing.T) {
	// Write config to file
	i := &Config{
		ServerPort:    "8000",
		DBHost:        "localhost",
		DBPort:        "3306",
		DBName:        "rmx",
		DBUser:        "rmx",
		DBPassword:    "password",
		RedisHost:     "localhost",
		RedisPort:     "6379",
		RedisPassword: "password",
	}

	if err := i.WriteToFile(false); err != nil {
		t.Fatal(err)
	}

	// Read config from file
	o, err := ScanConfigFile(false)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(i, o) {
		t.Fatalf("expected:\n%+v\ngot:\n%+v", i, o)
	}
}
