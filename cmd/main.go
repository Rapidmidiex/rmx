package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rog-golang-buddies/rapidmidiex/www"
)

func main() {
	s := www.NewService(chi.NewMux())
	log.Println("http://localhost:8888")
	log.Fatalln(http.ListenAndServe(":8888", s))
}
