package main

import (
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/spacecowboy/feeder-sync/sql"
)

func main() {
	conn := os.Getenv("FEEDER_SYNC_POSTGRES_CONN")

	if conn == "" {
		log.Fatal("FEEDER_SYNC_POSTGRES_CONN environment variable not set")
	}

	if err := runMigrations(conn); err != nil {
		log.Fatalf("Failed to run migrations: %s", err.Error())
	}
}

func runMigrations(dbURL string) error {
	log.Println("Loading migrations...")
	d, err := iofs.New(sql.MigrationsFS, "schema")
	if err != nil {
		return err
	}

	log.Println("Creating migrator...")
	m, err := migrate.NewWithSourceInstance("iofs", d, dbURL)
	if err != nil {
		return err
	}

	log.Println("Running migrations as necessary...")
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	log.Println("All migrations applied successfully")

	return nil
}
