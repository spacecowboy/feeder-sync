package migrations

import (
	"fmt"
	"log"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/spacecowboy/feeder-sync/sql"
)

// RunMigrations runs all migrations in the schema directory
// against the database at the given URL
// The database URL should be in the format postgres://user:password@localhost:5432/dbname
// In fact any scheme is accepted, but the pgx driver is used
func RunMigrations(dbURL string) error {
	log.Println("Loading migrations...")
	d, err := iofs.New(sql.MigrationsFS, "schema")
	if err != nil {
		return err
	}

	log.Println("Creating migrator...")
	m, err := migrate.NewWithSourceInstance("iofs", d, fixDbUrl(dbURL))
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

// Ensures protocol is pgx5://
// which is necessary for the pgx driver to be used
// with the golang-migrate library
func fixDbUrl(dbURL string) string {
	p := strings.SplitN(dbURL, "://", 2)

	return fmt.Sprintf("pgx5://%s", p[1])
}
