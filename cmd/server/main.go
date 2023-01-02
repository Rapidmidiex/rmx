package main

import (
	"log"
	"os"

	"github.com/rapidmidiex/rmx/config"
	"github.com/rapidmidiex/rmx/internal/cmds"
)

func main() {
	rmxEnv := os.Getenv("RMX_ENV")
	isDev := false
	if rmxEnv == "development" {
		isDev = true
	}
	cfg, err := config.LoadConfigFromEnv(isDev)
	if err != nil {
		log.Fatalf("Could load config: %v", err)
	}

	err = commands.StartServer(cfg)
	if err != nil {
		log.Fatalf("Could not start server: %v", err)
	}
}
