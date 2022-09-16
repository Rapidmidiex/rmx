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
	// if err := loadConfig(); err != nil {
	// 	return err
	// }

	port := getEnv("PORT", "8888")

	sCtx, cancel := signal.NotifyContext(context.Background(), syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	c := cors.Options{
		AllowedOrigins:   []string{"http://localhost:" + port},
		AllowCredentials: true,
		AllowedMethods:   []string{http.MethodGet},
		AllowedHeaders:   []string{"Origin", "Content-Type", "Accept", "Authorization"},
	}

	srv := http.Server{
		Addr:         ":" + port,
		Handler:      cors.New(c).Handler(www.NewService(chi.NewMux())),
		ReadTimeout:  10 * time.Second,  // max time to read request from the client
		WriteTimeout: 10 * time.Second,  // max time to write response to the client
		IdleTimeout:  120 * time.Second, // max time for connections using TCP Keep-Alive
		BaseContext:  func(_ net.Listener) context.Context { return sCtx },
	}

	g, gCtx := errgroup.WithContext(sCtx)

	g.Go(func() error {
		// Run the server
		log.Printf("App server starting on %s", srv.Addr)
		return srv.ListenAndServe()
	})

	g.Go(func() error {
		<-gCtx.Done()
		return srv.Shutdown(context.Background())
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
