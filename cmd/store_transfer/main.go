package main

import (
	"log"
	"os"

	"github.com/spacecowboy/feeder-sync/internal/store/postgres"
	"github.com/spacecowboy/feeder-sync/internal/store/sqlite"
	"github.com/spacecowboy/feeder-sync/internal/store/store_transfer"
)

func main() {
	// Open sqlite database
	log.Println("Opening sqlite store")
	sqlite, err := sqlite.New("./sqlite.db")
	if err != nil {
		log.Fatalf("Failed to open sqlite store: %q", err)
	}

	if err := sqlite.RunMigrations("file://./migrations_sqlite"); err != nil {
		log.Fatalf("Sqlite migration failed: %q", err)
	}

	// Get postgres connection string from environment variable
	conn := os.Getenv("FEEDER_SYNC_POSTGRES_CONN")

	if conn == "" {
		log.Fatal("FEEDER_SYNC_POSTGRES_CONN environment variable not set")
	}

	// Open postgres database
	log.Println("Opening postgres store")
	psql, err := postgres.New(conn)
	if err != nil {
		log.Fatalf("Failed to open postgres store: %q", err)
	}

	if err := psql.RunMigrations("file://./migrations_postgres"); err != nil {
		log.Fatalf("Postgres migration failed: %q", err)
	}

	log.Println("Transferring data from sqlite to postgres")
	if err := store_transfer.MoveBetweenStores(&sqlite, &psql); err != nil {
		log.Fatalf("Failed to transfer data: %q", err)
	}

	log.Println("Data transfer complete")
}
