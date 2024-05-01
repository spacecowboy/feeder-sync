package postgres

import (
	crand "crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/spacecowboy/feeder-sync/internal/store"
)

type PostgresStore struct {
	db *sql.DB
}

func New(conn string) (PostgresStore, error) {
	db, err := sql.Open("postgres", conn)
	if err != nil {
		return PostgresStore{}, err
	}

	return PostgresStore{
		db: db,
	}, nil
}

func (s *PostgresStore) RunMigrations(path string) error {
	driver, err := postgres.WithInstance(s.db, &postgres.Config{})
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

// TODO move to common, same for sqlite
func randomLegacySyncCode() (string, error) {
	bytes := make([]byte, 30)
	if _, err := crand.Read(bytes); err != nil {
		return "", err
	}
	syncCode := fmt.Sprintf("feed%s", hex.EncodeToString(bytes))

	if got := len(syncCode); got != 64 {
		log.Printf("code was %d long", got)
		return "", fmt.Errorf("Code was %d long not 64", got)
	}
	return syncCode, nil
}

func RegisterNewUser(db *sql.DB, deviceName string) (store.UserDevice, error) {
	userDevice := store.UserDevice{
		UserId:         uuid.New(),
		DeviceId:       uuid.New(),
		DeviceName:     deviceName,
		LegacyDeviceId: rand.Int63(),
		LastSeen:       time.Now().UnixMilli(),
	}

	legacySyncCode, err := randomLegacySyncCode()

	if err != nil {
		log.Printf("could not generate sync code: %s", err.Error())
		return userDevice, err
	}

	userDevice.LegacySyncCode = legacySyncCode

	// Insert user
	err = db.QueryRow(
		"INSERT INTO users (user_id, legacy_sync_code) VALUES ($1, $2) RETURNING db_id",
		userDevice.UserId,
		userDevice.LegacySyncCode,
	).Scan(&userDevice.UserDbId)
	if err != nil {
		log.Printf("could not insert user: %s", err.Error())
		return userDevice, err
	}

	return AddDeviceToUser(db, userDevice)
}

func AddDeviceToChain(db *sql.DB, userId uuid.UUID, deviceName string) (store.UserDevice, error) {
	var userDbId int64
	row := db.QueryRow("SELECT db_id FROM users WHERE user_id = $1 limit 1", userId)
	if err := row.Scan(&userDbId); err != nil {
		if err == sql.ErrNoRows {
			return store.UserDevice{}, errors.New("user not found")
		}
		return store.UserDevice{}, err
	}

	legacySyncCode, err := randomLegacySyncCode()

	if err != nil {
		return store.UserDevice{}, err
	}

	userDevice := store.UserDevice{
		UserDbId:       userDbId,
		UserId:         uuid.New(),
		DeviceId:       uuid.New(),
		DeviceName:     deviceName,
		LastSeen:       time.Now().UnixMilli(),
		LegacySyncCode: legacySyncCode,
		LegacyDeviceId: rand.Int63(),
	}

	return AddDeviceToUser(db, userDevice)
}

func AddDeviceToChainWithLegacy(db *sql.DB, syncCode string, deviceName string) (store.UserDevice, error) {
	var userDbId int64
	row := db.QueryRow("SELECT db_id FROM users WHERE legacy_sync_code = $1 limit 1", syncCode)
	if err := row.Scan(&userDbId); err != nil {
		if err == sql.ErrNoRows {
			return store.UserDevice{}, errors.New("user not found")
		}
		return store.UserDevice{}, err
	}

	userDevice := store.UserDevice{
		UserDbId:       userDbId,
		UserId:         uuid.New(),
		DeviceId:       uuid.New(),
		DeviceName:     deviceName,
		LastSeen:       time.Now().UnixMilli(),
		LegacySyncCode: syncCode,
		LegacyDeviceId: rand.Int63(),
	}

	return AddDeviceToUser(db, userDevice)
}

func AddDeviceToUser(db *sql.DB, userDevice store.UserDevice) (store.UserDevice, error) {
	var dbId int64
	// Insert device
	if err := db.QueryRow(
		"INSERT INTO devices (device_id, legacy_device_id, device_name, last_seen, user_db_id) VALUES ($1, $2, $3, $4, $5) RETURNING db_id",
		userDevice.DeviceId,
		userDevice.LegacyDeviceId,
		userDevice.DeviceName,
		userDevice.LastSeen,
		userDevice.UserDbId,
	).Scan(&dbId); err != nil {
		log.Printf("could not insert device: %s", err.Error())
		return userDevice, err
	}

	return userDevice, nil
}

func EnsureMigration(db *sql.DB, syncCode string, deviceId int64, deviceName string) (int64, error) {
	if len(syncCode) != 64 {
		return 0, fmt.Errorf("not a 64 char synccode: %q", syncCode)
	}

	var userCount int64
	var deviceCount int64
	var userDbId int64
	var deviceDbId int64

	// Insert user
	if err := db.QueryRow("INSERT INTO users (user_id, legacy_sync_code) VALUES ($1, $2) RETURNING db_id", uuid.New(), syncCode).Scan(&userDbId); err != nil {
		if !strings.Contains(err.Error(), "idx_users_legacy_sync_code") {
			return userCount + deviceCount, fmt.Errorf("insert user: %v", err)
		}

		row := db.QueryRow("SELECT db_id FROM users WHERE legacy_sync_code = $1 limit 1", syncCode)
		if err := row.Scan(&userDbId); err != nil {
			if err == sql.ErrNoRows {
				return userCount + deviceCount, fmt.Errorf("no user with syncCode %q", syncCode)
			}
			return userCount + deviceCount, fmt.Errorf("could not find user: %v", err)
		}
	} else {
		userCount++
		log.Printf("Migrated user %s", syncCode)
	}

	// Insert device
	if err := db.QueryRow(
		"INSERT INTO devices (device_id, legacy_device_id, device_name, last_seen, user_db_id) VALUES ($1, $2, $3, $4, $5) RETURNING db_id",
		uuid.New(),
		deviceId,
		deviceName,
		time.Now().UnixMilli(),
		userDbId,
	).Scan(&deviceDbId); err != nil {
		if !strings.Contains(err.Error(), "idx_devices_user_db_id_legacy_device_id") {
			return userCount + deviceCount, fmt.Errorf("insert device: %v", err)
		}
	} else {
		deviceCount++
		log.Printf("Migrated device for user %d, %s", deviceId, syncCode)
	}

	return userCount + deviceCount, nil
}

func GetLegacyDevice(db *sql.DB, syncCode string, deviceId int64) (store.UserDevice, error) {

	row := db.QueryRow(
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
		where legacy_sync_code = $1 and legacy_device_id = $2
		limit 1
		`,
		syncCode,
		deviceId,
	)

	userDevice := store.UserDevice{}
	if err := row.Scan(&userDevice.UserDbId, &userDevice.UserId, &userDevice.DeviceId, &userDevice.DeviceName, &userDevice.LegacySyncCode, &userDevice.LegacyDeviceId, &userDevice.LastSeen); err != nil {
		if err == sql.ErrNoRows {
			return userDevice, store.ErrNoSuchDevice
		}
		return userDevice, err
	}

	if err := row.Err(); err != nil {
		return userDevice, err
	}

	return userDevice, nil
}

func UpdateLastSeenForDevice(db *sql.DB, device store.UserDevice) (int64, error) {
	result, err := db.Exec(
		`
		update devices
		  set last_seen = $1
			where user_db_id = $2 and device_id = $3
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

func GetArticles(db *sql.DB, userId uuid.UUID, sinceMillis int64) ([]store.Article, error) {
	rows, err := db.Query(
		`
		select
		  user_id,
			read_time,
			identifier,
			updated_at
		from articles
		inner join users on articles.user_db_id = users.db_id
		where users.user_id = $1 and updated_at > $2
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

func AddLegacyArticle(db *sql.DB, userDbId int64, identifier string) error {
	now := time.Now().UnixMilli()
	_, err := db.Exec(
		`insert into articles (user_db_id, identifier, read_time, updated_at) values($1, $2 ,$3, $4)`,
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

func RemoveDeviceWithLegacy(db *sql.DB, userDbId int64, legacyDeviceId int64) (int64, error) {
	result, err := db.Exec(
		`
		delete from devices
		  where user_db_id = $1 and legacy_device_id = $2
		`,
		userDbId,
		legacyDeviceId,
	)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func GetDevices(db *sql.DB, userId uuid.UUID) ([]store.UserDevice, error) {
	rows, err := db.Query(
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
		where user_id = $1
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

func GetLegacyFeeds(db *sql.DB, userId uuid.UUID) (store.LegacyFeeds, error) {
	row := db.QueryRow(
		`
		select
			user_id,
			content_hash,
			content,
			etag
		from legacy_feeds
		inner join users on legacy_feeds.user_db_id = users.db_id
		where user_id = $1
		`,
		userId,
	)

	feeds := store.LegacyFeeds{}

	if err := row.Scan(&feeds.UserId, &feeds.ContentHash, &feeds.Content, &feeds.Etag); err != nil {
		if err == sql.ErrNoRows {
			return feeds, store.ErrNoFeeds
		} else {
			return feeds, err
		}
	}
	return feeds, nil
}

func UpdateLegacyFeeds(db *sql.DB, userDbId int64, contentHash int64, content string, etag string) (int64, error) {
	result, err := db.Exec(
		`
		insert into
		  legacy_feeds (user_db_id, content_hash, content, etag)
			values($1, $2, $3, $4)
			on conflict (user_db_id) do
			  update set
				  content_hash = excluded.content_hash,
					content = excluded.content,
					etag = excluded.etag
		`,
		userDbId,
		contentHash,
		content,
		etag,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
