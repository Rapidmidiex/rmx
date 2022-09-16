package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/cors"
	"golang.org/x/sync/errgroup"

	"github.com/rog-golang-buddies/rapidmidiex/www"
)

func run() error {
	port := getEnv("PORT", "8888")

	c := cors.Options{
		AllowedOrigins:   []string{"http://localhost:" + port},
		AllowCredentials: true,
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodDelete, http.MethodPatch},
		AllowedHeaders:   []string{"Origin", "Content-Type", "Accept", "Authorization"},
	}

	mux := chi.NewMux()
	service := www.NewService(mux)

	h := cors.New(c).Handler(service)

	serverCtx, serverStop := signal.NotifyContext(
		context.Background(),
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	defer serverStop()

	server := http.Server{
		Addr:         ":" + port,
		Handler:      h,
		ReadTimeout:  10 * time.Second,  // max time to read request from the client
		WriteTimeout: 10 * time.Second,  // max time to write response to the client
		IdleTimeout:  120 * time.Second, // max time for connections using TCP Keep-Alive
		BaseContext: func(_ net.Listener) context.Context {
			return serverCtx
		},
	}

	g, gCtx := errgroup.WithContext(serverCtx)
	g.Go(func() error {
		// Run the server
		return server.ListenAndServe()
	})
	g.Go(func() error {
		<-gCtx.Done()
		return server.Shutdown(context.Background())
	})

	if err := g.Wait(); err != nil {
		log.Printf("exit reason: %s \n", err)
	}

	return nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
