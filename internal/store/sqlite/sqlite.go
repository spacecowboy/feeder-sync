package sqlite

import (
	"database/sql"
	"errors"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/spacecowboy/feeder-sync/internal/store"
	_ "modernc.org/sqlite"
)

type SqliteStore struct {
	db *sql.DB
}

func New(dbPath string) (SqliteStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return SqliteStore{}, err
	}

	return SqliteStore{
		db: db,
	}, nil
}

func (s SqliteStore) Close() error {
	return s.db.Close()
}

func (s SqliteStore) RunMigrations(path string) error {
	driver, err := sqlite.WithInstance(s.db, &sqlite.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance(path, "feeder", driver)
	if err != nil {
		return err
	}

	return m.Up()
}

func (s SqliteStore) RegisterNewUser(deviceName string) (store.UserDevice, error) {
	return store.UserDevice{}, errors.New("TODO")
}

func (s SqliteStore) AddDeviceToChain(userId uuid.UUID, deviceName string) (store.UserDevice, error) {
	return store.UserDevice{}, errors.New("TODO")
}

func (s SqliteStore) AddDeviceToChainWithLegacy(syncCode string, deviceName string) (store.UserDevice, error) {
	return store.UserDevice{}, errors.New("TODO")
}

func (s SqliteStore) EnsureMigration(syncCode string, deviceId int64, deviceName string) error {
	return errors.New("TODO")
}
