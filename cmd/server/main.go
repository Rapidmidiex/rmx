package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httplog"
	jams "github.com/rapidmidiex/rmx/internal/jam/http"
	"github.com/rapidmidiex/rmx/static"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	q := make(chan os.Signal, 1)
	signal.Notify(q, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-q
		cancel()
	}()

	if err := run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) (err error) {
	mux := chi.NewMux()

	logger := newLogger(AppName)
	mux.Use(httplog.RequestLogger(logger))
	{
		mux.Handle("/", http.RedirectHandler("/jams", http.StatusMovedPermanently))
		mux.Mount("/jams", jams.NewService())
		mux.Handle("/static/*", &static.Static{Prefix: "/static/"})
	}

	s := http.Server{
		Addr:         Addr,
		Handler:      mux,
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
		// figure out way to do better config
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	sErr := make(chan error, 1)
	go func() {
		sErr <- s.ListenAndServe()
	}()

	select {
	case err := <-sErr:
		return fmt.Errorf("main error: starting server: %w", err)
	case <-ctx.Done():
		const timeout = 5 * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		err := s.Shutdown(ctx)
		if err != nil {
			err = s.Close()
		}

		if err != nil {
			return fmt.Errorf("main error: could not stop server gracefully : %w", err)
		}

		return nil
	}
}