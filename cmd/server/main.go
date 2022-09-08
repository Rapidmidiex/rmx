package main

import (
	"log"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rog-golang-buddies/rapidmidiex"
	"github.com/rog-golang-buddies/rapidmidiex/api"
)

func main() {
	err := rmx.LoadConfig()
	if err != nil {
		log.Fatalf("failed to read config: %v", err.Error())
	}

	jamService := api.JamService{}

	server := api.Server{
		Port:   ":8080",
		Router: chi.NewMux(),
	}

	server.Router.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.Logger)
		r.Route("/jam", func(r chi.Router) {
			r.Post("/new", jamService.NewSession)
			r.Get("/{session_id}/join", jamService.JoinSession)
		})
	})

	log.Println("starting the server")
	server.ServeHTTP()
}
