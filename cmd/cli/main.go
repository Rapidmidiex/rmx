package main

import (
	"log"
	"os"
	"time"

	"github.com/rapidmidiex/rmx/internal/cmd"
	"github.com/urfave/cli/v2"
)

func main() {
	if err := initCLI().Run(os.Args); err != nil {
		log.Fatalln(err)
	}
}

func initCLI() *cli.App {
	c := &cli.App{
		EnableBashCompletion: true,
		Name:                 "rmx",
		Usage:                "RapidMidiEx Server CLI",
		Version:              cmds.Version,
		Compiled:             time.Now().UTC(),
		Action:               cmds.GetVersion,
		Flags:                cmds.Flags,
		Commands:             cmds.Commands,
	}

	return c
}
