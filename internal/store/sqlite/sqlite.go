package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
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

func (s *SqliteStore) Close() error {
	return s.db.Close()
}

func (s *SqliteStore) RunMigrations(path string) error {
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

func (s *SqliteStore) RegisterNewUser(deviceName string) (store.UserDevice, error) {
	return store.UserDevice{}, errors.New("TODO")
}

func (s *SqliteStore) AddDeviceToChain(userId uuid.UUID, deviceName string) (store.UserDevice, error) {
	return store.UserDevice{}, errors.New("TODO")
}

func (s *SqliteStore) AddDeviceToChainWithLegacy(syncCode string, deviceName string) (store.UserDevice, error) {
	return store.UserDevice{}, errors.New("TODO")
}

func (s *SqliteStore) EnsureMigration(syncCode string, deviceId int64, deviceName string) (int64, error) {
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
		log.Printf("Migrated user %s", syncCode)
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
		log.Printf("Migrated device for user %d, %s", deviceId, syncCode)
	}

	return userCount + deviceCount, nil
}

func (s *SqliteStore) GetLegacyDevice(syncCode string, deviceId int64) (store.UserDevice, error) {

	row := s.db.QueryRow(
		`
		select
		  users.db_id,
		  user_id,
			device_id,
			device_name,
			legacy_sync_code,
			legacy_device_id,
			last_seen
		from devices
		inner join users on devices.user_db_id = users.db_id
		where legacy_sync_code = ? and legacy_device_id = ?
		limit 1
		`,
		syncCode,
		deviceId,
	)

	userDevice := store.UserDevice{}
	if err := row.Scan(&userDevice.UserDbId, &userDevice.UserId, &userDevice.DeviceId, &userDevice.DeviceName, &userDevice.LegacySyncCode, &userDevice.LegacyDeviceId, &userDevice.LastSeen); err != nil {
		return userDevice, err
	}

	if err := row.Err(); err != nil {
		return userDevice, err
	}

	return userDevice, nil
}

func (s *SqliteStore) UpdateLastSeenForDevice(device store.UserDevice) (int64, error) {
	result, err := s.db.Exec(
		`
		update devices
		  set last_seen = ?
			where user_db_id = ? and device_id = ?
		`,
		time.Now().UnixMilli(),
		device.UserDbId,
		device.DeviceId,
	)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func (s *SqliteStore) GetArticles(userId uuid.UUID, sinceMillis int64) ([]store.Article, error) {
	rows, err := s.db.Query(
		`
		select
		  user_id,
			read_time,
			identifier,
			updated_at
		from articles
		inner join users on articles.user_db_id = users.db_id
		where users.user_id = ? and updated_at > ?
		`,
		userId,
		sinceMillis,
	)

	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	defer rows.Close()

	var articles []store.Article

	for rows.Next() {
		var article store.Article
		if err := rows.Scan(&article.UserId, &article.ReadTime, &article.Identifier, &article.UpdatedAt); err != nil {
			log.Println(err.Error())
			return articles, err
		}
		articles = append(articles, article)
	}
	if err = rows.Err(); err != nil {
		log.Println(err.Error())
		return articles, err
	}

	return articles, nil
}

func (s *SqliteStore) AddLegacyArticle(userDbId int64, identifier string) error {
	now := time.Now().UnixMilli()
	_, err := s.db.Exec(
		`insert into articles (user_db_id, identifier, read_time, updated_at) values(?, ? ,?, ?)`,
		userDbId,
		identifier,
		now,
		now,
	)
	if err != nil {
		if !strings.Contains(err.Error(), "UNIQUE constraint failed: articles.user_db_id, articles.identifier") {
			return err
		}
	}

	return nil
}

func (s *SqliteStore) RemoveDeviceWithLegacy(userDbId int64, legacyDeviceId int64) (int64, error) {
	result, err := s.db.Exec(
		`
		delete from devices
		  where user_db_id = ? and legacy_device_id = ?
		`,
		userDbId,
		legacyDeviceId,
	)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func (s *SqliteStore) GetDevices(userId uuid.UUID) ([]store.UserDevice, error) {
	rows, err := s.db.Query(
		`
		select
		  users.db_id,
		  user_id,
			device_id,
			device_name,
			legacy_sync_code,
			legacy_device_id,
			last_seen
		from devices
		inner join users on devices.user_db_id = users.db_id
		where user_id = ?
		`,
		userId,
	)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []store.UserDevice

	for rows.Next() {
		var userDevice store.UserDevice
		if err := rows.Scan(&userDevice.UserDbId, &userDevice.UserId, &userDevice.DeviceId, &userDevice.DeviceName, &userDevice.LegacySyncCode, &userDevice.LegacyDeviceId, &userDevice.LastSeen); err != nil {
			log.Println(err.Error())
			return devices, err
		}

		devices = append(devices, userDevice)
	}
	if err = rows.Err(); err != nil {
		log.Println(err.Error())
		return devices, err
	}

	return devices, nil
}
