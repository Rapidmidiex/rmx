package cmd

import (
	"context"
	"database/sql"
	"log"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	"github.com/go-chi/chi/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/rapidmidiex/rmx/internal/cmd/config"
	jamHTTP "github.com/rapidmidiex/rmx/internal/jam/http"
	"github.com/rapidmidiex/rmx/internal/sessions"

	authHTTP "github.com/rapidmidiex/rmx/internal/auth/http"
	jamDB "github.com/rapidmidiex/rmx/internal/jam/store"

	"github.com/rs/cors"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"
)

func run(dev bool) func(cCtx *cli.Context) error {
	return func(cCtx *cli.Context) error {
		return serve(config.LoadFromEnv())
	}
}

func serve(cfg *config.Config) error {
	sCtx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	defer cancel()

	c := cors.Options{
		AllowedOrigins:   []string{"http://localhost:*", "http://127.0.0.1:*"}, // ? band-aid, needs to change to a flag
		AllowCredentials: true,
		AllowedMethods:   []string{http.MethodGet, http.MethodPost},
		AllowedHeaders:   []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposedHeaders:   []string{"Location"},
		Debug:            cfg.Dev,
	}

	conn, err := sql.Open("postgres", cfg.Store.DatabaseURL)
	if err != nil {
		return err
	}

	authOpts := []authHTTP.Option{
		authHTTP.WithContext(sCtx),
		authHTTP.WithProvider(
			cfg.Auth.Domain,
			cfg.Auth.ClientID,
			cfg.Auth.ClientSecret,
			cfg.Auth.LoginCallbackURL,
			cfg.Auth.LogoutCallbackURL,
		),
		authHTTP.WithServiceURLs(cfg.Auth.RedirectURL, cfg.Auth.LogoutURL),
	}

	sessHandler, err := sessions.New("_rmx_session", 24*30*time.Hour, []byte(cfg.Auth.SessionKey))
	if err != nil {
		return err
	}

	authService := authHTTP.New(authOpts...)

	mux := chi.NewMux()
	mux.Use(sessHandler)
	mux.Route("/v0", func(r chi.Router) {
		r.Mount("/jams", newJamService(sCtx, conn))
		r.Mount("/auth", authService)
	})

	srv := http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: cors.New(c).Handler(mux),
		// max time to read request from the client
		ReadTimeout: 10 * time.Second,
		// max time to write response to the client
		WriteTimeout: 10 * time.Second,
		// max time for connections using TCP Keep-Alive
		IdleTimeout: 120 * time.Second,
		BaseContext: func(_ net.Listener) context.Context { return sCtx },
		ErrorLog:    log.Default(),
	}

	g, gCtx := errgroup.WithContext(sCtx)

	g.Go(func() error {
		// Run the server
		srv.ErrorLog.Printf("rmx server starting on %s", srv.Addr)
		return srv.ListenAndServe()
	})

	g.Go(func() error {
		<-gCtx.Done()
		return srv.Shutdown(context.Background())
	})

	return g.Wait()
}

// StartServer starts the RMX application.
func StartServer(cfg *config.Config) error {
	return serve(cfg)
}

func newJamService(ctx context.Context, conn *sql.DB) *jamHTTP.Service {
	jamDB := jamDB.New(conn)
	jamHTTP := jamHTTP.New(ctx, jamDB)
	return jamHTTP
}
