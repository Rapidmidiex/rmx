package commands

import (
	"errors"

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
		Action:      runProd,
		Flags:       Flags,
	},
	{
		Name:        "dev",
		Category:    "run",
		Aliases:     []string{"d"},
		Description: "Starts the server in development mode",
		Action: func(cCtx *cli.Context) error {
			return runDev()
		},
		Flags: Flags,
	},
}
