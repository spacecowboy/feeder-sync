package postgres

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
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/spacecowboy/feeder-sync/internal/store"
)

type PostgresStore struct {
	Db *sql.DB
}

// verify interface implementation
var _ store.DataStore = &PostgresStore{}

func New(conn string) (PostgresStore, error) {
	db, err := sql.Open("postgres", conn)
	if err != nil {
		return PostgresStore{}, err
	}

	return PostgresStore{
		Db: db,
	}, nil
}

func (s *PostgresStore) PingContext(ctx context.Context) error {
	return s.Db.PingContext(ctx)
}

func (s *PostgresStore) Close() error {
	return s.Db.Close()
}

func (s *PostgresStore) RunMigrations(path string) error {
	driver, err := postgres.WithInstance(s.Db, &postgres.Config{})
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

func (s *PostgresStore) RegisterNewUser(deviceName string) (store.UserDevice, error) {
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
	err = s.Db.QueryRow(
		"INSERT INTO users (user_id, legacy_sync_code) VALUES ($1, $2) RETURNING db_id",
		userDevice.UserId,
		userDevice.LegacySyncCode,
	).Scan(&userDevice.UserDbId)
	if err != nil {
		log.Printf("could not insert user: %s", err.Error())
		return userDevice, err
	}

	return s.AddDeviceToUser(userDevice)
}

func (s *PostgresStore) AddDeviceToChain(userId uuid.UUID, deviceName string) (store.UserDevice, error) {
	var userDbId int64
	row := s.Db.QueryRow("SELECT db_id FROM users WHERE user_id = $1 limit 1", userId)
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

func (s *PostgresStore) AddDeviceToChainWithLegacy(syncCode string, deviceName string) (store.UserDevice, error) {
	var userDbId int64
	row := s.Db.QueryRow("SELECT db_id FROM users WHERE legacy_sync_code = $1 limit 1", syncCode)
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

func (s *PostgresStore) AddDeviceToUser(userDevice store.UserDevice) (store.UserDevice, error) {
	var dbId int64
	// Insert device
	if err := s.Db.QueryRow(
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

func (s *PostgresStore) EnsureMigration(syncCode string, deviceId int64, deviceName string) (int64, error) {
	if len(syncCode) != 64 {
		return 0, fmt.Errorf("not a 64 char synccode: %q", syncCode)
	}

	var userCount int64
	var deviceCount int64
	var userDbId int64
	var deviceDbId int64

	// Insert user
	if err := s.Db.QueryRow("INSERT INTO users (user_id, legacy_sync_code) VALUES ($1, $2) RETURNING db_id", uuid.New(), syncCode).Scan(&userDbId); err != nil {
		if !strings.Contains(err.Error(), "idx_users_legacy_sync_code") {
			return userCount + deviceCount, fmt.Errorf("insert user: %v", err)
		}

		row := s.Db.QueryRow("SELECT db_id FROM users WHERE legacy_sync_code = $1 limit 1", syncCode)
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
	if err := s.Db.QueryRow(
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

func (s *PostgresStore) GetLegacyDevicesEtag(syncCode string) (string, error) {
	row := s.Db.QueryRow(
		`
		select
		  sha256(
				convert_to(
					string_agg(device_name, '' order by device_name),
					 'UTF8'))
		from devices
		inner join users on devices.user_db_id = users.db_id
		where legacy_sync_code = $1
		group by user_db_id
		`,
		syncCode,
	)

	var etag string

	if err := row.Scan(&etag); err != nil {
		if err == sql.ErrNoRows {
			return "", store.ErrNoSuchDevice
		} else {
			return "", err
		}
	}

	if etag == "" {
		return "", store.ErrNoSuchDevice
	}
	return etag, nil
}

func (s *PostgresStore) GetLegacyDevice(syncCode string, deviceId int64) (store.UserDevice, error) {

	row := s.Db.QueryRow(
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

func (s *PostgresStore) UpdateLastSeenForDevice(device store.UserDevice) (int64, error) {
	result, err := s.Db.Exec(
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

func (s *PostgresStore) GetArticles(userId uuid.UUID, sinceMillis int64) ([]store.Article, error) {
	rows, err := s.Db.Query(
		`
		select
		  user_id,
			read_time,
			identifier,
			updated_at
		from articles
		inner join users on articles.user_db_id = users.db_id
		where users.user_id = $1 and updated_at > $2
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

func (s *PostgresStore) AddLegacyArticle(userDbId int64, identifier string) error {
	now := time.Now().UnixMilli()
	_, err := s.Db.Exec(
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

func (s *PostgresStore) RemoveDeviceWithLegacy(userDbId int64, legacyDeviceId int64) (int64, error) {
	result, err := s.Db.Exec(
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

func (s *PostgresStore) GetDevices(userId uuid.UUID) ([]store.UserDevice, error) {
	rows, err := s.Db.Query(
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

func (s PostgresStore) GetLegacyFeedsEtag(userId uuid.UUID) (string, error) {
	row := s.Db.QueryRow(
		`
		select
			etag
		from legacy_feeds
		inner join users on legacy_feeds.user_db_id = users.db_id
		where user_id = $1
		`,
		userId,
	)

	var etag string

	if err := row.Scan(&etag); err != nil {
		if err == sql.ErrNoRows {
			return "", store.ErrNoFeeds
		} else {
			return "", err
		}
	}
	return etag, nil
}

func (s *PostgresStore) GetLegacyFeeds(userId uuid.UUID) (store.LegacyFeeds, error) {
	row := s.Db.QueryRow(
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

func (s *PostgresStore) UpdateLegacyFeeds(userDbId int64, contentHash int64, content string, etag string) (int64, error) {
	result, err := s.Db.Exec(
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

func (s *PostgresStore) TransferUsersToStore(toStore store.DataStore) error {
	rows, err := s.Db.Query(
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

	var user store.User

	// print a u for every 100 users
	var user_count int = -1
	var printed bool = false

	for rows.Next() {
		if err := rows.Scan(&user.UserDbId, &user.UserId, &user.LegacySyncCode); err != nil {
			return err
		}

		toStore.AcceptUser(&user)

		user_count++
		if user_count%100 == 0 {
			printed = true
			fmt.Printf("u")
		}
	}
	if printed {
		fmt.Printf("\n")
	}
	if err = rows.Err(); err != nil {
		return err
	}
	return nil
}

func (s *PostgresStore) AcceptUser(user *store.User) error {
	// Insert user
	_, err := s.Db.Exec(
		"INSERT INTO users (db_id, user_id, legacy_sync_code) VALUES ($1, $2, $3)",
		user.UserDbId,
		user.UserId,
		user.LegacySyncCode,
	)
	return err
}

func (s *PostgresStore) TransferDevicesToStore(toStore store.DataStore) error {
	rows, err := s.Db.Query(
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

	var userDevice store.UserDevice

	// print a d for every 100 users
	var item_count int = -1
	var printed bool = false

	for rows.Next() {
		if err := rows.Scan(&userDevice.UserDbId, &userDevice.DeviceId, &userDevice.DeviceName, &userDevice.LegacyDeviceId, &userDevice.LastSeen); err != nil {
			return err
		}

		toStore.AcceptDevice(&userDevice)

		item_count++
		if item_count%100 == 0 {
			printed = true
			fmt.Printf("d")
		}
	}
	if printed {
		fmt.Printf("\n")
	}
	if err = rows.Err(); err != nil {
		return err
	}
	return nil
}

func (s *PostgresStore) AcceptDevice(device *store.UserDevice) error {
	// Insert device
	_, err := s.Db.Exec(
		"INSERT INTO devices (device_id, legacy_device_id, device_name, last_seen, user_db_id) VALUES ($1, $2, $3, $4, $5)",
		device.DeviceId,
		device.LegacyDeviceId,
		device.DeviceName,
		device.LastSeen,
		device.UserDbId,
	)
	return err
}

func (s *PostgresStore) TransferArticlesToStore(toStore store.DataStore) error {
	rows, err := s.Db.Query(
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

	var article store.Article

	// print a a for every 100 users
	var item_count int = -1
	var printed bool = false

	for rows.Next() {
		if err := rows.Scan(&article.ReadTime, &article.Identifier, &article.UpdatedAt, &article.UserDbId); err != nil {
			return err
		}

		toStore.AcceptArticle(&article)

		item_count++
		if item_count%100 == 0 {
			printed = true
			fmt.Printf("a")
		}
	}
	if printed {
		fmt.Printf("\n")
	}
	if err = rows.Err(); err != nil {
		return err
	}
	return nil
}

func (s *PostgresStore) AcceptArticle(article *store.Article) error {
	// Insert article
	_, err := s.Db.Exec(
		`insert into articles (user_db_id, identifier, read_time, updated_at) values($1, $2, $3, $4)`,
		article.UserDbId,
		article.Identifier,
		article.ReadTime,
		article.UpdatedAt,
	)
	return err
}

func (s *PostgresStore) TransferLegacyFeedsToStore(toStore store.DataStore) error {
	rows, err := s.Db.Query(
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

	var feeds store.LegacyFeeds

	// print a f for every 100 users
	var item_count int = -1
	var printed bool = false

	for rows.Next() {
		if err := rows.Scan(&feeds.ContentHash, &feeds.Content, &feeds.Etag, &feeds.UserDbId); err != nil {
			return err
		}

		toStore.AcceptLegacyFeeds(&feeds)

		item_count++
		if item_count%100 == 0 {
			printed = true
			fmt.Printf("f")
		}
	}
	if printed {
		fmt.Printf("\n")
	}

	if err = rows.Err(); err != nil {
		return err
	}
	return nil
}

func (s *PostgresStore) AcceptLegacyFeeds(feeds *store.LegacyFeeds) error {
	// Insert legacy feeds
	_, err := s.Db.Exec(
		`
		insert into
		  legacy_feeds (user_db_id, content_hash, content, etag)
			values($1, $2, $3, $4)
		`,
		feeds.UserDbId,
		feeds.ContentHash,
		feeds.Content,
		feeds.Etag,
	)
	return err
}
