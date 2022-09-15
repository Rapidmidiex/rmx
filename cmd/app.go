package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rog-golang-buddies/rapidmidiex/www"
	"github.com/rs/cors"
)

type App struct {
	srv *http.Server
	l   *log.Logger
}

func NewApp(addr string, h http.Handler, read, write, idle time.Duration) *App {
	c := cors.Options{
		AllowedOrigins:   []string{"http://localhost" + addr},
		AllowCredentials: true,
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodDelete, http.MethodPatch},
		AllowedHeaders:   []string{"Origin", "Content-Type", "Accept", "Authorization"},
	}

	a := &App{
		srv: &http.Server{
			Addr:         addr,
			Handler:      cors.New(c).Handler(h),
			ReadTimeout:  read,
			WriteTimeout: write,
			IdleTimeout:  idle,
		},
		l: log.Default(),
	}

	return a
}

func DefaultApp() *App {
	mux := chi.NewMux()
	h := www.NewService(mux)

	return NewApp(":8888", h, 10*time.Second, 10*time.Second, 120*time.Second)
}

func (a App) Shutdown() error {
	return a.srv.Shutdown(context.Background())
}

func (a *App) ListenAndServe() error {
	return a.srv.ListenAndServe()
}
