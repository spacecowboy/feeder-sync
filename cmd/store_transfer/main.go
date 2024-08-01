package main

import (
	"log"
	"os"

	"github.com/spacecowboy/feeder-sync/internal/store/postgres"
	"github.com/spacecowboy/feeder-sync/internal/store/store_transfer"
)

func main() {
	// Get old postgres connection string from environment variable
	conn_old := os.Getenv("FEEDER_SYNC_POSTGRES_CONN_OLD")

	if conn_old == "" {
		log.Fatal("FEEDER_SYNC_POSTGRES_CONN_OLD environment variable not set")
	}

	// Open old postgres database
	log.Println("Opening old postgres store")
	psql_old, err := postgres.New(conn_old)
	if err != nil {
		log.Fatalf("Failed to open old postgres store: %q", err)
	}

	if err := psql_old.RunMigrations("file://./migrations_postgres"); err != nil {
		log.Fatalf("Postgres migration failed: %q", err)
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
	if err := store_transfer.MoveBetweenStores(&psql_old, &psql); err != nil {
		log.Fatalf("Failed to transfer data: %q", err)
	}

	log.Println("Data transfer complete")
}
