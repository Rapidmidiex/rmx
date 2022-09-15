package main

import (
	"log"
)

func main() {
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}

func run() error {
	app := DefaultApp()
	app.l.Printf("Running... http://localhost%s/", app.srv.Addr)

	return app.ListenAndServe()
}
