package cmd

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/manifoldco/promptui"
	"github.com/rapidmidiex/rmx/internal/auth"
	"github.com/rapidmidiex/rmx/internal/cmd/internal/config"
	jamHTTP "github.com/rapidmidiex/rmx/internal/jam/http"

	authHTTP "github.com/rapidmidiex/rmx/internal/auth/http"
	"github.com/rapidmidiex/rmx/internal/auth/provider/github"
	"github.com/rapidmidiex/rmx/internal/auth/provider/google"
	jamDB "github.com/rapidmidiex/rmx/internal/jam/postgres"

	"github.com/rs/cors"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"
)

func run(dev bool) func(cCtx *cli.Context) error {
	var f = func(cCtx *cli.Context) error {
		// check if a config file exists and use that
		c, err := config.ScanConfigFile() // set dev mode true/false
		if err != nil {
			return errors.New("failed to scan config file")
		}
		if c != nil {
			configPrompt := promptui.Prompt{
				Label:     "A config file was found. do you want to use it?",
				IsConfirm: true,
				Default:   "y",
			}

			validateConfirm := func(s string) error {
				if len(s) == 1 && strings.Contains("YyNn", s) ||
					configPrompt.Default != "" && len(s) == 0 {
					return nil
				}
				return errors.New(`invalid input (you can only use "y" or "n")`)
			}

			configPrompt.Validate = validateConfirm

			result, err := configPrompt.Run()
			if err != nil {
				if strings.ToLower(result) != "n" {
					return err
				}
			}

			if strings.ToLower(result) == "y" {
				return serve(c)
			}

		}
		return nil
	}
	return f
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

	// ? should this defined within the instantiation of a new service
	c := cors.Options{
		AllowedOrigins:   []string{"*"}, // ? band-aid, needs to change to a flag
		AllowCredentials: true,
		AllowedMethods:   []string{http.MethodGet, http.MethodPost},
		AllowedHeaders:   []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposedHeaders:   []string{"Location"},
		Debug:            cfg.Dev,
	}

	/* FIXME */
	/* START SERVICES BLOCK */
	dbURL := fmt.Sprintf(
		"postgres://%s:%s@%s/%s?sslmode=disable",
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBHost,
		cfg.DBName,
	)

	// Just use connection string if available
	if cfg.DBURL != "" {
		dbURL = cfg.DBURL
	}

	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		return err
	}

	// I don't like this pattern
	githubCfg := &auth.ProviderCfg{ClientID: cfg.GithubClientID, ClientSecret: cfg.GithubClientSecret}
	googleCfg := &auth.ProviderCfg{ClientID: cfg.GoogleClientID, ClientSecret: cfg.GoogleClientSecret}

	// What should we do with the errors?
	// is this the right way to handle them?
	// or maybe newAuthService should handle the errors itself?
	authService, err := newAuthService(sCtx, githubCfg, googleCfg)
	if err != nil {
		return err
	}

	mux := chi.NewMux()
	mux.Route("/v0", func(r chi.Router) {
		r.Mount("/jams", newJamService(sCtx, conn))
		r.Mount("/auth", authService)
	})

	for _, r := range mux.Routes() {
		s, _ := json.MarshalIndent(r, "", "\t")
		fmt.Println(string(s))
	}

	/* START SERVICES BLOCK */
	srv := http.Server{
		Addr:    ":" + cfg.ServerPort,
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
		srv.ErrorLog.Printf("App server starting on %s", srv.Addr)
		return srv.ListenAndServe()
	})

	g.Go(func() error {
		<-gCtx.Done()
		return srv.Shutdown(context.Background())
	})

	// if err := g.Wait(); err != nil {
	// 	log.Printf("exit reason: %s \n", err)
	// }

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

// TODO: find a better way to pass provider config
func newAuthService(ctx context.Context, githubCfg, googleCfg *auth.ProviderCfg) (*authHTTP.Service, error) {
	githubService, err := github.NewGithub(githubCfg)
	if err != nil {
		return nil, err
	}

	googleService, err := google.NewGoogle(googleCfg)
	if err != nil {
		return nil, err
	}

	authHTTP := authHTTP.New(ctx, githubService, googleService)

	return authHTTP, nil
}
