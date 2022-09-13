package main

import (
	"log"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	rmx "github.com/rog-golang-buddies/rapidmidiex"
	"github.com/rog-golang-buddies/rapidmidiex/api"
	"github.com/spf13/viper"
)

func main() {
	err := rmx.LoadConfig()
	if err != nil {
		log.Fatalf("failed to read config: %v", err.Error())
	}

	jamService := api.JamService{}

	server := api.Server{
		Port:   ":" + viper.GetString("PORT"),
		Router: chi.NewMux(),
	}

	server.Router.Route("/ws/v1", func(r chi.Router) {
		r.Use(middleware.Logger)
		r.Route("/jam", func(r chi.Router) {
			r.Get("/new", jamService.Connect)
		})
	})

	log.Println("starting the server")
	server.ServeHTTP()
}
