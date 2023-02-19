package cmd

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/manifoldco/promptui"
	"github.com/rapidmidiex/rmx/internal/cmd/internal/config"
	jamHTTP "github.com/rapidmidiex/rmx/internal/jam/http"
	jamDB "github.com/rapidmidiex/rmx/internal/jam/postgres"

	"github.com/rs/cors"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"
)

func run(dev bool) func(cCtx *cli.Context) error {
	var f = func(cCtx *cli.Context) error {
		templates := &promptui.PromptTemplates{
			Prompt:  "{{ . }} ",
			Valid:   "{{ . | green }} ",
			Invalid: "{{ . | red }} ",
			Success: "{{ . | bold }} ",
		}

		// Server Port
		validateNumber := func(v string) error {
			if _, err := strconv.ParseUint(v, 0, 0); err != nil {
				return errors.New("invalid number")
			}

			return nil
		}

		validateString := func(v string) error {
			if !(len(v) > 0) {
				return errors.New("invalid string")
			}

			return nil
		}

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

		// Server Port
		serverPortPrompt := promptui.Prompt{
			Label:     "Server Port",
			Validate:  validateNumber,
			Templates: templates,
		}

		serverPort, err := serverPortPrompt.Run()
		if err != nil {
			return err
		}

		// DB Host
		dbHostPrompt := promptui.Prompt{
			Label:     "Postgres Database host",
			Validate:  validateString,
			Templates: templates,
		}

		dbHost, err := dbHostPrompt.Run()
		if err != nil {
			return err
		}

		// DB Port
		dbPortPrompt := promptui.Prompt{
			Label:     "Postgres Database port",
			Validate:  validateNumber,
			Templates: templates,
		}

		dbPort, err := dbPortPrompt.Run()
		if err != nil {
			return err
		}

		// DB Name
		dbNamePrompt := promptui.Prompt{
			Label:     "Postgres Database name",
			Validate:  validateString,
			Templates: templates,
		}

		dbName, err := dbNamePrompt.Run()
		if err != nil {
			return err
		}

		// DB User
		dbUserPrompt := promptui.Prompt{
			Label:     "Postgres Database user",
			Validate:  validateString,
			Templates: templates,
		}

		dbUser, err := dbUserPrompt.Run()
		if err != nil {
			return err
		}

		// DB Password
		dbPasswordPrompt := promptui.Prompt{
			Label:     "Postgres Database password",
			Validate:  validateString,
			Templates: templates,
			Mask:      '*',
		}

		dbPassword, err := dbPasswordPrompt.Run()
		if err != nil {
			return err
		}

		// Redis Host
		redisHostPrompt := promptui.Prompt{
			Label:     "Redis host",
			Validate:  validateString,
			Templates: templates,
		}

		redisHost, err := redisHostPrompt.Run()
		if err != nil {
			return err
		}

		// Redis Port
		redisPortPrompt := promptui.Prompt{
			Label:     "Redis port",
			Validate:  validateNumber,
			Templates: templates,
		}

		redisPort, err := redisPortPrompt.Run()
		if err != nil {
			return err
		}

		// Redis Password
		redisPasswordPrompt := promptui.Prompt{
			Label:     "Redis password",
			Validate:  validateString,
			Templates: templates,
			Mask:      '*',
		}

		redisPassword, err := redisPasswordPrompt.Run()
		if err != nil {
			return err
		}

		c = &config.Config{
			ServerPort:    serverPort,
			DBHost:        dbHost,
			DBPort:        dbPort,
			DBName:        dbName,
			DBUser:        dbUser,
			DBPassword:    dbPassword,
			RedisHost:     redisHost,
			RedisPort:     redisPort,
			RedisPassword: redisPassword,
			Dev:           dev,
		}

		// prompt to save the config to a file
		configPrompt := promptui.Prompt{
			Label:     "Do you want to write the config to a file? (NOTE: this will rewrite the config file)",
			IsConfirm: true,
			Default:   "n",
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
			if err := c.WriteToFile(); err != nil {
				return err
			}
		}

		return serve(c)
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
	jamHTTP := newJamService(sCtx, conn)

	/* START SERVICES BLOCK */
	srv := http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: cors.New(c).Handler(jamHTTP),
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
