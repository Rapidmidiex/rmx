package api

import (
	"context"
	"log"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/rs/cors"
	"golang.org/x/sync/errgroup"
)

type Server struct {
	Host   string
	Port   string
	Router *chi.Mux
}

func (s *Server) ServeHTTP() {
	// "*" shouldn't be used as AllowedOrigins
	c := cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://127.0.0.1:3000"},
		AllowCredentials: true,
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodDelete, http.MethodPatch},
		AllowedHeaders:   []string{"Origin", "Content-Type", "Accept", "Authorization"},
	}
	h := cors.New(c).Handler(s.Router)

	serverCtx, serverStop := signal.NotifyContext(
		context.Background(),
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	defer serverStop()

	server := http.Server{
		Addr:         s.Port,
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
}

func NewServer() *Server {
	jamService := JamService{}
	s := new(Server)

	s.Router = chi.NewMux()

	s.Router.Route("/ws/v1", func(r chi.Router) {
		r.Use(middleware.Logger)
		r.Get("/jam", jamService.Connect)
	})

	return s
}
