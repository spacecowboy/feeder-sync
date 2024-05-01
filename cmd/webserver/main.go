package main

import (
	"log"
	"net/http"

	"github.com/spacecowboy/feeder-sync/internal/server"
)

func main() {
	server, err := server.NewServerWithSqlite()
	if err != nil {
		log.Fatalf("Failed to create server: %q", err)
	}
	defer func() {
		if err := server.Close(); err != nil {
			return
		}
	}()

	addr := ":34217"
	log.Printf("Serving on %q", addr)
	log.Fatal(http.ListenAndServe(addr, server))
}
