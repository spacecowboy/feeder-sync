package main

import (
	"log"
	"os"

	"github.com/spacecowboy/feeder-sync/internal/migrations"
)

func main() {
	conn := os.Getenv("FEEDER_SYNC_POSTGRES_CONN")

	if conn == "" {
		log.Fatal("FEEDER_SYNC_POSTGRES_CONN environment variable not set")
	}

	if err := migrations.RunMigrations(conn); err != nil {
		log.Fatalf("Failed to run migrations: %s", err.Error())
	}
}
