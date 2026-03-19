package main

import (
	"log"

	"github.com/spacecowboy/feeder-sync/internal/config"
	"github.com/spacecowboy/feeder-sync/internal/migrations"
)

func main() {
	conn, err := config.GetDatabaseConn()
	if err != nil {
		log.Fatal(err.Error())
	}

	if err := migrations.RunMigrations(conn); err != nil {
		log.Fatalf("migrations: %s", err.Error())
	}
}
