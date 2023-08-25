package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

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

	if err := m.Up(); err != migrate.ErrNoChange {
		return err
	}

	return nil
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

func (s SqliteStore) EnsureMigration(syncCode string, deviceId int64, deviceName string) (int64, error) {
	if len(syncCode) != 64 {
		return 0, fmt.Errorf("Not a 64 char synccode: %q", syncCode)
	}

	var userCount int64
	var deviceCount int64
	var userDbId int64

	// Insert user
	result, err := s.db.Exec("INSERT INTO users (user_id, legacy_sync_code) VALUES (?, ?)", uuid.New(), syncCode)
	if err != nil {
		if !strings.Contains(err.Error(), "constraint failed: users.legacy_sync_code") {
			return userCount + deviceCount, fmt.Errorf("insert user: %v", err)
		}

		row := s.db.QueryRow("SELECT db_id FROM users WHERE legacy_sync_code = ?", syncCode)
		if err := row.Scan(&userDbId); err != nil {
			if err == sql.ErrNoRows {
				return userCount + deviceCount, fmt.Errorf("no user with syncCode %q", syncCode)
			}
			return userCount + deviceCount, fmt.Errorf("could not find user: %v", err)
		}
	} else {
		userCount, err = result.RowsAffected()
		if err != nil {
			return userCount + deviceCount, fmt.Errorf("insert user2: %v", err)
		}
		userDbId, err = result.LastInsertId()
		if err != nil {
			return userCount + deviceCount, fmt.Errorf("insert user2: %v", err)
		}
	}

	// Insert device
	result, err = s.db.Exec(
		"INSERT INTO devices (device_id, legacy_device_id, device_name, last_seen, user_db_id) VALUES (?, ?, ?, ?, ?)",
		uuid.New(),
		deviceId,
		deviceName,
		time.Now().UnixMilli(),
		userDbId,
	)

	if err != nil {
		if !strings.Contains(err.Error(), "constraint failed: devices.user_db_id, devices.legacy_device_id") {
			return userCount + deviceCount, fmt.Errorf("insert user: %v", err)
		}
	} else {
		deviceCount, err = result.RowsAffected()
		if err != nil {
			return userCount + deviceCount, fmt.Errorf("insert user2: %v", err)
		}
	}

	return userCount + deviceCount, nil
}
