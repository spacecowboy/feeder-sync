package main

import (
	"log"
	"net/http"

	"github.com/spacecowboy/feeder-sync/server"
)

func main() {
	server, err := server.NewServer()
	if err != nil {
		log.Fatalf("Failed to create server: %q", err)
	}
	defer func() {
		if err := server.Close(); err != nil {
			return
		}
	}()

	addr := ":5000"
	log.Printf("Serving on %q", addr)
	log.Fatal(http.ListenAndServe(addr, &server))
}
