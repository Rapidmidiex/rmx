package main

import (
	"log"
	"net/http"

	rmx "github.com/rog-golang-buddies/rapidmidiex"
	"github.com/rog-golang-buddies/rapidmidiex/api"
	"github.com/spf13/viper"
)

func main() {
	err := rmx.LoadConfig()
	if err != nil {
		log.Fatalf("failed to read config: %v", err.Error())
	}

	server := api.NewServer()

	port := ":" + viper.GetString("PORT")
	log.Printf("starting the server on %s%s\n", server.Host, port)

	log.Fatal(http.ListenAndServe(port, server.Router))
}
