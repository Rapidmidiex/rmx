package main

import (
	"log"

	"github.com/rapidmidiex/rmx/internal/cmd"
	"github.com/rapidmidiex/rmx/internal/cmd/config"
)

func main() {
	if err := cmd.StartServer(config.LoadFromEnv()); err != nil {
		log.Fatalf("rmx: couldn't start server\n%v", err)
	}
}
