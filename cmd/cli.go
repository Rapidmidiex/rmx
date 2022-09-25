package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
)

type config struct {
	Port int `json:"port"`
}

var (
	ErrInvalidPort = errors.New("invalid port number")
)

func initCLI() *cli.App {
	flags := []cli.Flag{
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

	c := &cli.App{
		Name:     "rmx",
		Usage:    "RapidMidiEx Server CLI",
		Version:  "v0.0.1",
		Compiled: time.Now().UTC(),
		Action: func(*cli.Context) error {
			return nil
		},
		Before: altsrc.InitInputSourceWithContext(flags, altsrc.NewYamlSourceFromFlagFunc("load")),
		Commands: []*cli.Command{
			{
				Name:        "start",
				Category:    "run",
				Aliases:     []string{"s"},
				Description: "Starts the server in production mode.",
				Action: func(cCtx *cli.Context) error {
					port := cCtx.Int("port")
					fmt.Println(port)
					if port < 0 {
						return ErrInvalidPort
					}

					cfg := &config{
						Port: port,
					}

					return run(cfg)
				},
				Flags: flags,
			},
			{
				Name:        "dev",
				Category:    "run",
				Aliases:     []string{"d"},
				Description: "Starts the server in development mode",
				Action: func(cCtx *cli.Context) error {
					return nil
				},
				Flags: flags,
			},
		},
	}

	return c
}
