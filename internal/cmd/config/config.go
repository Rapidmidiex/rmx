package config

import (
	"errors"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type ServerConfig struct {
	Port string
}

type StoreConfig struct {
	DatabaseURL string
}

type AuthConfig struct {
	Domain       string
	Audience     []string
	ClientID     string
	ClientSecret string
	CallbackURL  string
	RedirectURL  string
	SessionKey   string
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
	authDomain := readEnvStr("AUTH_DOMAIN")
	authAudience := readEnvStrArray("AUTH_AUDIENCE")
	clientID := readEnvStr("AUTH_CLIENT_ID")
	clientSecret := readEnvStr("AUTH_CLIENT_SECRET")
	callbackURL := readEnvStr("AUTH_CALLBACK_URL")
	redirectURL := readEnvStr("AUTH_REDIRECT_URL")
	sessionKey := readEnvStr("AUTH_SESSION_KEY")

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
			Domain:       authDomain,
			Audience:     authAudience,
			ClientID:     clientID,
			ClientSecret: clientSecret,
			CallbackURL:  callbackURL,
			RedirectURL:  redirectURL,
			SessionKey:   sessionKey,
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

func readEnvStrArray(key string) []string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("rmx: no value assigned for key [%s]", key)
	}

	return strings.Split(v, " ")
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
