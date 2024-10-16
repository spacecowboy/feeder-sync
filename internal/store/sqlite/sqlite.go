package sqlite

import (
	"context"
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
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/spacecowboy/feeder-sync/internal/store"
	_ "modernc.org/sqlite"
)

type SqliteStore struct {
	db *sql.DB
}

// verify interface implementation
var _ store.DataStore = &SqliteStore{}

func New(dbPath string) (SqliteStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return SqliteStore{}, err
	}

	db.SetMaxOpenConns(1)

	return SqliteStore{
		db: db,
	}, nil
}

func (s *SqliteStore) Close() error {
	return s.db.Close()
}

func (s *SqliteStore) PingContext(ctx context.Context) error {
	return s.db.PingContext(ctx)
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

func randomLegacySyncCode() (string, error) {
	bytes := make([]byte, 30)
	if _, err := crand.Read(bytes); err != nil {
		return "", err
	}
	syncCode := fmt.Sprintf("feed%s", hex.EncodeToString(bytes))

	if got := len(syncCode); got != 64 {
		log.Printf("code was %d long", got)
		return "", errors.New(fmt.Sprintf("Code was %d long not 64", got))
	}
	return syncCode, nil
}

func (s *SqliteStore) RegisterNewUser(deviceName string) (store.UserDevice, error) {
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
	result, err := s.db.Exec(
		"INSERT INTO users (user_id, legacy_sync_code) VALUES (?, ?)",
		userDevice.UserId,
		userDevice.LegacySyncCode,
	)
	if err != nil {
		log.Printf("could not insert user: %s", err.Error())
		return userDevice, err
	}

	userDbId, err := result.LastInsertId()
	if err != nil {
		log.Printf("could get last inserted id: %s", err.Error())
		return userDevice, err
	}

	userDevice.UserDbId = userDbId

	return s.AddDeviceToUser(userDevice)
}

func (s *SqliteStore) AddDeviceToChain(userId uuid.UUID, deviceName string) (store.UserDevice, error) {
	var userDbId int64
	row := s.db.QueryRow("SELECT db_id FROM users WHERE user_id = ? limit 1", userId)
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

	return s.AddDeviceToUser(userDevice)
}

func (s *SqliteStore) AddDeviceToChainWithLegacy(syncCode string, deviceName string) (store.UserDevice, error) {
	var userDbId int64
	row := s.db.QueryRow("SELECT db_id FROM users WHERE legacy_sync_code = ? limit 1", syncCode)
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

	return s.AddDeviceToUser(userDevice)
}

func (s *SqliteStore) AddDeviceToUser(userDevice store.UserDevice) (store.UserDevice, error) {
	// Insert device
	result, err := s.db.Exec(
		"INSERT INTO devices (device_id, legacy_device_id, device_name, last_seen, user_db_id) VALUES (?, ?, ?, ?, ?)",
		userDevice.DeviceId,
		userDevice.LegacyDeviceId,
		userDevice.DeviceName,
		userDevice.LastSeen,
		userDevice.UserDbId,
	)

	if err != nil {
		log.Printf("could not insert device: %s", err.Error())
		return userDevice, err
	}

	count, err := result.RowsAffected()
	if err != nil {
		log.Printf("could not get rows affected: %s", err.Error())
		return userDevice, err
	}

	if count < 1 {
		log.Printf("wrong rows affected: %d", count)
		return userDevice, errors.New(fmt.Sprintf("expected one inserted row but was %d", count))
	}

	return userDevice, nil
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

		row := s.db.QueryRow("SELECT db_id FROM users WHERE legacy_sync_code = ? limit 1", syncCode)
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
			return userCount + deviceCount, fmt.Errorf("insert user3: %v", err)
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
			return userCount + deviceCount, fmt.Errorf("insert device: %v", err)
		}
	} else {
		deviceCount, err = result.RowsAffected()
		if err != nil {
			return userCount + deviceCount, fmt.Errorf("insert device2: %v", err)
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

func (s *SqliteStore) GetLegacyDevicesEtag(syncCode string) (string, error) {
	// Not supported yet, so just do a random string
	return randomLegacySyncCode()
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
		order by read_time desc
		limit 1000
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

func (s SqliteStore) GetLegacyFeedsEtag(userId uuid.UUID) (string, error) {
	row := s.db.QueryRow(
		`
		select
			etag
		from legacy_feeds
		inner join users on legacy_feeds.user_db_id = users.db_id
		where user_id = ?
		`,
		userId,
	)

	var etag string

	if err := row.Scan(&etag); err != nil {
		if err == sql.ErrNoRows {
			return etag, store.ErrNoFeeds
		} else {
			return etag, err
		}
	}
	return etag, nil
}

func (s *SqliteStore) GetLegacyFeeds(userId uuid.UUID) (store.LegacyFeeds, error) {
	row := s.db.QueryRow(
		`
		select
			user_id,
			content_hash,
			content,
			etag
		from legacy_feeds
		inner join users on legacy_feeds.user_db_id = users.db_id
		where user_id = ?
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

func (s *SqliteStore) UpdateLegacyFeeds(userDbId int64, contentHash int64, content string, etag string) (int64, error) {
	result, err := s.db.Exec(
		`
		insert into
		  legacy_feeds (user_db_id, content_hash, content, etag)
			values(?, ?, ?, ?)
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

func (s *SqliteStore) TransferUsersToStore(toStore store.DataStore) error {
	rows, err := s.db.Query(
		`
		select
		  db_id,
		  user_id,
			legacy_sync_code
		from users
		`,
	)

	if err != nil {
		return err
	}
	defer rows.Close()

	var count int
	var user store.User

	for rows.Next() {
		if err := rows.Scan(&user.UserDbId, &user.UserId, &user.LegacySyncCode); err != nil {
			return err
		}
		log.Println("User: ", count)

		toStore.AcceptUser(&user)
		count++
	}
	if err = rows.Err(); err != nil {
		return err
	}
	return nil
}

func (s *SqliteStore) AcceptUser(user *store.User) error {
	// Insert user
	_, err := s.db.Exec(
		"INSERT INTO users (db_id, user_id, legacy_sync_code) VALUES (?, ?, ?)",
		user.UserDbId,
		user.UserId,
		user.LegacySyncCode,
	)
	return err
}

func (s *SqliteStore) TransferDevicesToStore(toStore store.DataStore) error {
	rows, err := s.db.Query(
		`
		select
		  user_db_id,
			device_id,
			device_name,
			legacy_device_id,
			last_seen
		from devices
		`,
	)

	if err != nil {
		return err
	}
	defer rows.Close()

	var count int
	var userDevice store.UserDevice

	for rows.Next() {
		if err := rows.Scan(&userDevice.UserDbId, &userDevice.DeviceId, &userDevice.DeviceName, &userDevice.LegacyDeviceId, &userDevice.LastSeen); err != nil {
			return err
		}
		log.Println("Device: ", count)

		toStore.AcceptDevice(&userDevice)
		count++
	}
	if err = rows.Err(); err != nil {
		return err
	}
	return nil
}

func (s *SqliteStore) AcceptDevice(device *store.UserDevice) error {
	// Insert device
	_, err := s.db.Exec(
		"INSERT INTO devices (device_id, legacy_device_id, device_name, last_seen, user_db_id) VALUES (?, ?, ?, ?, ?)",
		device.DeviceId,
		device.LegacyDeviceId,
		device.DeviceName,
		device.LastSeen,
		device.UserDbId,
	)
	return err
}

func (s *SqliteStore) TransferArticlesToStore(toStore store.DataStore) error {
	rows, err := s.db.Query(
		`
		select
		  read_time,
			identifier,
			updated_at,
			user_db_id
		from articles
		`,
	)

	if err != nil {
		return err
	}
	defer rows.Close()

	var count int
	var article store.Article

	for rows.Next() {
		if err := rows.Scan(&article.ReadTime, &article.Identifier, &article.UpdatedAt, &article.UserDbId); err != nil {
			return err
		}
		log.Println("Article: ", count)

		toStore.AcceptArticle(&article)
		count++
	}
	if err = rows.Err(); err != nil {
		return err
	}
	return nil
}

func (s *SqliteStore) AcceptArticle(article *store.Article) error {
	// Insert article
	_, err := s.db.Exec(
		`insert into articles (user_db_id, identifier, read_time, updated_at) values(?, ? ,?, ?)`,
		article.UserDbId,
		article.Identifier,
		article.ReadTime,
		article.UpdatedAt,
	)
	return err
}

func (s *SqliteStore) TransferLegacyFeedsToStore(toStore store.DataStore) error {
	rows, err := s.db.Query(
		`
		select
		  content_hash,
			content,
			etag,
			user_db_id
		from legacy_feeds
		`,
	)

	if err != nil {
		return err
	}
	defer rows.Close()

	var count int
	var feeds store.LegacyFeeds

	for rows.Next() {
		if err := rows.Scan(&feeds.ContentHash, &feeds.Content, &feeds.Etag, &feeds.UserDbId); err != nil {
			return err
		}

		log.Println("LegacyFeeds: ", count)
		toStore.AcceptLegacyFeeds(&feeds)
		count++
	}

	if err = rows.Err(); err != nil {
		return err
	}
	return nil
}

func (s *SqliteStore) AcceptLegacyFeeds(feeds *store.LegacyFeeds) error {
	// Insert legacy feeds
	_, err := s.db.Exec(
		`
		insert into
		  legacy_feeds (user_db_id, content_hash, content, etag)
			values(?, ?, ?, ?)
		`,
		feeds.UserDbId,
		feeds.ContentHash,
		feeds.Content,
		feeds.Etag,
	)
	return err
}
