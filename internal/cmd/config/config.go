package config

import (
	"errors"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type ServerConfig struct {
	Port string
}

type StoreConfig struct {
	DatabaseURL string
}

type AuthConfig struct {
	ClientID     string
	ClientSecret string
}

type Config struct {
	Server ServerConfig
	Store  StoreConfig
	Auth   AuthConfig
	Dev    bool
}

const rmxEnvPath = "rmx.env"

func LoadFromEnv() *Config {
	if _, err := os.Stat(rmxEnvPath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil
		}
	} else {
		if err := godotenv.Load(rmxEnvPath); err != nil {
			log.Fatalf("rmx: couldn't read env\n%v", err)
		}
	}

	// server
	serverPort := readEnvStr("SERVER_PORT")

	// store
	databaseURL := readEnvStr("DATABASE_URL")

	// auth
	clientID := readEnvStr("GOOGLE_CLIENT_ID")
	clientSecret := readEnvStr("GOOGLE_CLIENT_SECRET")

	// env
	dev := readEnvBool("DEV")

	return &Config{
		Server: ServerConfig{
			Port: serverPort,
		},
		Store: StoreConfig{
			DatabaseURL: databaseURL,
		},
		Auth: AuthConfig{
			ClientID:     clientID,
			ClientSecret: clientSecret,
		},
		Dev: dev,
	}
}

func readEnvStr(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("rmx: no value assigned for key [%s]", key)
	}
	return v
}

/*
func readEnvInt(key string) (int, error) {
	s, err := readEnvStr(key)
	if err != nil {
		return 0, err
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	return v, nil
}
*/

func readEnvBool(key string) bool {
	s := readEnvStr(key)
	v, err := strconv.ParseBool(s)
	if err != nil {
		log.Fatalf("rmx: couldn't parse (bool) value from key [%s]", key)
	}
	return v
}
