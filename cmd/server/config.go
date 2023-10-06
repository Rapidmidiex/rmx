package main

import (
	"errors"
	"os"

	"github.com/go-chi/httplog"
	"github.com/rs/zerolog"
)

var (
	AppName, Addr string
)

func init() {
	if Addr = ":"+os.Getenv("PORT"); Addr == ":" {
		Addr = ":8000"
	}

	if AppName = os.Getenv("FLY_APP_NAME"); AppName == "" {
		AppName = "web-app"
	}
}

func newLogger(serviceName string) zerolog.Logger {
	return httplog.NewLogger(serviceName, httplog.Options{Concise: true})
}

var (
	ErrStartServer    = errors.New("failed to start server")
	ErrShutdownServer = errors.New("failed to shutdown server")
)