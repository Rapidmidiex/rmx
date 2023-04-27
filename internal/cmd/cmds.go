package cmd

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/rapidmidiex/rmx/internal/cmd/internal/config"
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
)

var (
	ErrInvalidPort = errors.New("invalid port number")
)

var Flags = []cli.Flag{
	altsrc.NewIntFlag(&cli.IntFlag{
		Name:     "port",
		Value:    0,
		Usage:    "Defines the port which server should listen on",
		Required: false,
		Aliases:  []string{"p"},
		EnvVars:  []string{"PORT"},
	}),
	&cli.StringFlag{
		Name:    "load",
		Aliases: []string{"l"},
	},
}

var Commands = []*cli.Command{
	{
		Name:        "start",
		Category:    "run",
		Aliases:     []string{"s"},
		Description: "Starts the server in production mode.",
		Action:      run(false), // disable dev mode
		Flags:       Flags,
	},
	{
		Name:        "dev",
		Category:    "run",
		Aliases:     []string{"d"},
		Description: "Starts the server in development mode",
		Action:      run(true), // enable dev mode
		Flags:       Flags,
	},
}

// shouldn't be here
const Version = "v0.0.0-a.1"

func GetVersion(cCtx *cli.Context) error {
	_, err := fmt.Println("rmx version: " + Version)
	return err
}

// LoadConfigFromEnv creates a Config from environment variables.
func LoadConfigFromEnv(dev bool) (*config.Config, error) {
	serverPort := os.Getenv("PORT")

	pgURL := os.Getenv("POSTGRES_URL")
	if pgURL == "" {
		pgURL = os.Getenv("DATABASE_URL")
	}

	if dev {
		fmt.Printf("DATABASE_URL: %s\nPOSTGRES_URL: %s\n", os.Getenv("DATABASE_URL"), os.Getenv("POSTGRES_URL"))
	}

	pgParsed, err := url.Parse(pgURL)
	if err != nil && pgURL != "" {
		return nil, fmt.Errorf("invalid POSTGRES_URL env var: %q: %w", pgURL, err)
	}

	pgUser := pgParsed.User.Username()
	pgPassword, _ := pgParsed.User.Password()

	pgHost := pgParsed.Host
	pgPort := pgParsed.Port()
	pgName := strings.TrimPrefix(pgParsed.Path, "/")

	/*
		redisHost := os.Getenv("REDIS_HOST")
		redisPort := os.Getenv("REDIS_PORT")
		redisPassword := os.Getenv("REDIS_PASSWORD")
	*/

	return &config.Config{
		Port: serverPort,
		DB: config.DBConfig{
			Host:     pgHost,
			Port:     pgPort,
			Name:     pgName,
			User:     pgUser,
			Password: pgPassword,
		},
		Auth: config.AuthConfig{
			Google: config.GoogleConfig{
				ClientID:     "",
				ClientSecret: "",
			},
		},
		Dev: dev,
	}, nil
}
