package main

import (
	"log"

	rmx "github.com/rog-golang-buddies/rapidmidiex"
	"github.com/rog-golang-buddies/rapidmidiex/api"
	"github.com/spf13/viper"
)

func main() {
	err := rmx.LoadConfig()
	if err != nil {
		log.Fatalf("failed to read config: %v", err.Error())
	}

	server := api.NewServer(":" + viper.GetString("PORT"))

	log.Println("starting the server")
	server.ServeHTTP()
}
