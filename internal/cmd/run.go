package cmd

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/lib/pq"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/manifoldco/promptui"
	"github.com/rapidmidiex/rmx/internal/cmd/internal/config"
	jamHTTP "github.com/rapidmidiex/rmx/internal/jam/http"

	authHTTP "github.com/rapidmidiex/rmx/internal/auth/http"
	authDB "github.com/rapidmidiex/rmx/internal/auth/postgres"
	"github.com/rapidmidiex/rmx/internal/auth/provider"
	"github.com/rapidmidiex/rmx/internal/auth/provider/google"
	jamDB "github.com/rapidmidiex/rmx/internal/jam/postgres"

	"github.com/rs/cors"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"
)

func run(dev bool) func(cCtx *cli.Context) error {
	var f = func(cCtx *cli.Context) error {
		dbEnv := os.Getenv("DB_URL")
		// check if a config file exists and use that
		c, err := config.ScanConfigFile() // set dev mode true/false
		if err != nil {
			return errors.New("failed to scan config file")
		}
		if dbEnv != "" {
			dbParsed, err := url.Parse(dbEnv)
			if err != nil {
				return fmt.Errorf("invalid DB_URL env var: %q: %w", dbEnv, err)
			}

			dbHost := dbParsed.Host
			dbPort := dbParsed.Port()
			dbName := strings.TrimPrefix(dbParsed.Path, "/")

			dbUser := dbParsed.User.Username()
			dbPassword, _ := dbParsed.User.Password()

			c.DB = config.DBConfig{
				Host:     dbHost,
				Port:     dbPort,
				Name:     dbName,
				User:     dbUser,
				Password: dbPassword,
			}
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

	dbURL := fmt.Sprintf(
		"postgres://%s:%s@%s/%s?sslmode=disable",
		cfg.DB.User,
		cfg.DB.Password,
		cfg.DB.Host,
		cfg.DB.Name,
	)

	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		return err
	}

	if err := runMigrations(conn); err != nil && err != migrate.ErrNoChange {
		return err
	}

	googleCfg := google.New(
		cfg.Auth.Google.ClientID,
		cfg.Auth.Google.ClientSecret,
		[]byte(cfg.Auth.CookieHashKey),
		[]byte(cfg.Auth.CookieEncryptionKey),
	)

	mux := chi.NewMux()
	mux.Route("/v0", func(r chi.Router) {
		r.Mount("/jams", newJamService(sCtx, conn))
		r.Mount(
			"/auth",
			newAuthService(
				sCtx,
				fmt.Sprintf("http://localhost:%s/v0/auth", cfg.Port),
				conn,
				[]provider.Provider{
					googleCfg,
				},
				cfg.Auth.CookieHashKey,
				cfg.Auth.CookieEncryptionKey,
			),
		)
	})

	/* START SERVICES BLOCK */
	srv := http.Server{
		Addr:    ":" + cfg.Port,
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

	return g.Wait()
}

func runMigrations(conn *sql.DB) error {
	driver, err := postgres.WithInstance(conn, &postgres.Config{})
	if err != nil {
		return err
	}

	// run migrations on startup
	jamM, err := migrate.NewWithDatabaseInstance(
		"file://internal/jam/postgres/migration",
		"postgres",
		driver,
	)
	if err != nil {
		return err
	}
	if err := jamM.Up(); err != nil {
		return err
	}

	authM, err := migrate.NewWithDatabaseInstance(
		"file://internal/auth/postgres/migration",
		"postgres",
		driver,
	)
	if err != nil {
		return err
	}
	if err := authM.Up(); err != nil {
		return err
	}

	return nil
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
func newAuthService(
	ctx context.Context,
	baseURI string,
	conn *sql.DB,
	providers []provider.Provider,
	hashKey,
	encKey string,
) *authHTTP.Service {
	authDB := authDB.New(conn)
	authHTTP := authHTTP.New(ctx, baseURI, authDB, providers)
	return authHTTP
}
