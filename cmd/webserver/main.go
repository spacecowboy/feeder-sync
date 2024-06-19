package main

import (
	"log"
	"net/http"
	"os"

	"github.com/spacecowboy/feeder-sync/internal/server"
)

func main() {
	conn := os.Getenv("FEEDER_SYNC_POSTGRES_CONN")

	if conn == "" {
		log.Fatal("FEEDER_SYNC_POSTGRES_CONN environment variable not set")
	}

	server, err := server.NewServerWithPostgres(conn)

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
