package main

import (
	"log"
	"os"

	"github.com/rapidmidiex/rmx/internal/cmd"
)

func main() {
	rmxEnv := os.Getenv("RMX_ENV")
	isDev := false
	if rmxEnv == "development" {
		isDev = true
	}
	cfg, err := cmd.LoadConfigFromEnv(isDev)
	if err != nil {
		log.Fatalf("Could load config: %v", err)
	}

	err = cmd.StartServer(cfg)
	if err != nil {
		log.Fatalf("Could not start server: %v", err)
	}
}
