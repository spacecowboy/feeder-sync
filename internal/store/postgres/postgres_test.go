package postgres

import (
	"database/sql"
	"testing"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/migrators/golangmigrator"
	"github.com/spacecowboy/feeder-sync/internal/store"
)

// NewDB is a helper that returns an open connection to a unique and isolated
// test database, fully migrated and ready for you to query.
func NewDB(t *testing.T) *sql.DB {
	t.Helper()
	conf := pgtestdb.Config{
		DriverName: "postgres",
		User:       "postgres",
		Password:   "password",
		Host:       "localhost",
		Port:       "55432",
		Options:    "sslmode=disable",
	}
	migrator := golangmigrator.New("../../../migrations_postgres")

	return pgtestdb.New(t, conf, migrator)
}

func TestStoreRegister(t *testing.T) {
	db := NewDB(t)
	defer db.Close()

	t.Run("Register new user works", func(t *testing.T) {
		userDevice, err := RegisterNewUser(db, "devicename")

		if err != nil {
			t.Fatalf("error: %s", err.Error())
		}

		if userDevice.DeviceName != "devicename" {
			t.Errorf("wrong device name: %s", userDevice.DeviceName)
		}

		if userDevice.DeviceId == uuid.Nil {
			t.Errorf("bad device id: %s", userDevice.DeviceId)
		}

		if userDevice.UserId == uuid.Nil {
			t.Errorf("bad user id: %s", userDevice.UserId)
		}

		if userDevice.LegacySyncCode == "" {
			t.Errorf("bad LegacySyncCode id: %s", userDevice.LegacySyncCode)
		}

		if userDevice.LegacyDeviceId == 0 {
			t.Errorf("bad LegacyDeviceId id: %d", userDevice.LegacyDeviceId)
		}

		devices, err := GetDevices(db, userDevice.UserId)

		if err != nil {
			t.Fatalf("failed: %s", err.Error())
		}

		if got := len(devices); got != 1 {
			t.Fatalf("wrong number of devices: %d", got)
		}

		gotDevice := devices[0]

		if userDevice.DeviceName != gotDevice.DeviceName {
			t.Errorf("wrong device name: %s", userDevice.DeviceName)
		}

		if userDevice.DeviceId != gotDevice.DeviceId {
			t.Errorf("bad device id: %s", userDevice.DeviceId)
		}

		if userDevice.UserId != gotDevice.UserId {
			t.Errorf("bad user id: %s", userDevice.UserId)
		}

		if userDevice.LegacySyncCode != gotDevice.LegacySyncCode {
			t.Errorf("bad LegacySyncCode id: %s", userDevice.LegacySyncCode)
		}

		if userDevice.LegacyDeviceId != gotDevice.LegacyDeviceId {
			t.Errorf("bad LegacyDeviceId id: %d", userDevice.LegacyDeviceId)
		}
	})
}

func TestStoreAddToChain(t *testing.T) {
	db := NewDB(t)
	defer db.Close()

	userDevice, err := RegisterNewUser(db, "firstDevice")
	if err != nil {
		t.Fatalf("Failed: %s", err.Error())
	}

	t.Run("AddDeviceToChainWithLegacy no such user fails", func(t *testing.T) {
		_, err = AddDeviceToChainWithLegacy(db, "foo bar", "bla bla")
		if err == nil {
			t.Fatalf("Expected a failure")
		}

		if got := err.Error(); got != "user not found" {
			t.Fatalf("error should be %q, not %q", "user not found", got)
		}
	})

	t.Run("AddDeviceToChainWithLegacy succeeds", func(t *testing.T) {
		device, err := AddDeviceToChainWithLegacy(db, userDevice.LegacySyncCode, "secondDevice")
		if err != nil {
			t.Fatalf("failed: %s", err.Error())
		}

		if got := device.DeviceName; got != "secondDevice" {
			t.Errorf("Wrong device name: %s", got)
		}
	})

	t.Run("AddDeviceToChain no such user fails", func(t *testing.T) {
		_, err = AddDeviceToChain(db, uuid.New(), "bla bla")
		if err == nil {
			t.Fatalf("Expected a failure")
		}

		if got := err.Error(); got != "user not found" {
			t.Fatalf("error should be %q, not %q", "user not found", got)
		}
	})

	t.Run("AddDeviceToChain succeeds", func(t *testing.T) {
		device, err := AddDeviceToChain(db, userDevice.UserId, "otherDevice")
		if err != nil {
			t.Fatalf("failed: %s", err.Error())
		}

		if got := device.DeviceName; got != "otherDevice" {
			t.Errorf("Wrong device name: %s", got)
		}
	})
}

func TestStoreApi(t *testing.T) {
	db := NewDB(t)
	defer db.Close()

	legacySyncCode := "fa18973dd5889b64d8ec2a08ede95d94ee07d430d0d1b80b11bfd6a0375552c0"
	_, err := EnsureMigration(db, legacySyncCode, 1, "devicename")
	if err != nil {
		t.Fatalf("Got an error: %s", err.Error())
	}

	userDevice, err := GetLegacyDevice(db, legacySyncCode, 1)
	if err != nil {
		t.Fatalf("Got an error: %s", err.Error())
	}

	t.Run("Migration invalid synccode returns error", func(t *testing.T) {
		_, err := EnsureMigration(db, "tooshort", 1, "foo")

		if err == nil {
			t.Error("Expected an error")
		}
	})

	t.Run("Ensure migration works", func(t *testing.T) {
		var wantRows int64

		legacySyncCode := "ba18973dd5889b64d8ec2a08ede95d94ee07d430d0d1b80b11bfd6a0375552c0"
		wantRows = 2
		got, err := EnsureMigration(db, legacySyncCode, 66, "devicename")
		if err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}
		if wantRows != got {
			t.Errorf("Wanted %d rows, but was %d", wantRows, got)
		}

		// Add another device
		wantRows = 1
		got, err = EnsureMigration(db, legacySyncCode, 67, "devicename")
		if err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}
		if wantRows != got {
			t.Errorf("Wanted %d rows, but was %d", wantRows, got)
		}

		// Same device again
		wantRows = 0
		got, err = EnsureMigration(db, legacySyncCode, 67, "devicename")
		if err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}
		if wantRows != got {
			t.Errorf("Wanted %d rows, but was %d", wantRows, got)
		}

		// Ensure data is correct
		rows, err := db.Query(
			`select
			  device_id,
				legacy_device_id,
				device_name,
				last_seen,
				user_db_id
			from devices
			  where legacy_device_id = $1 or legacy_device_id = $2
			`,
			66,
			67,
		)
		if err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}
		defer rows.Close()

		// Loop through rows, using Scan to assign column data to struct fields.
		deviceCount := 0
		for rows.Next() {
			deviceCount += 1
			var deviceId uuid.UUID
			var legacyDeviceId int64
			var deviceName string
			var lastSeen int64
			var userDbId int64
			if err := rows.Scan(&deviceId, &legacyDeviceId, &deviceName, &lastSeen, &userDbId); err != nil {
				t.Fatalf("Got an error: %s", err.Error())
			}

			if userDbId != 2 {
				t.Errorf("Wrong userDbId: %v", userDbId)
			}
			if deviceName != "devicename" {
				t.Errorf("Didnt store devicename: %q", deviceName)
			}
			if lastSeen < 1 {
				t.Errorf("Bad value for lastSeen: %d", lastSeen)
			}

		}
		if err := rows.Err(); err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}

		if deviceCount != 2 {
			t.Errorf("Wanted 2, but device count: %d", deviceCount)
		}
	})

	t.Run("Write and get legacy articles", func(t *testing.T) {
		articles, err := GetArticles(db, userDevice.UserId, 0)
		if err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}

		if len(articles) != 0 {
			t.Fatalf("Expected no articles yet: %d", len(articles))
		}

		if err = AddLegacyArticle(db, userDevice.UserDbId, "first"); err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}

		// Now should get one
		articles, err = GetArticles(db, userDevice.UserId, 0)
		if err != nil {
			t.Fatalf("Got an error:%s", err.Error())
		}

		if len(articles) != 1 {
			t.Fatalf("Wrong number of articles: %d", len(articles))
		}

		article := articles[0]
		articles, err = GetArticles(db, userDevice.UserId, article.UpdatedAt)
		if err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}

		if len(articles) != 0 {
			t.Fatalf("Wrong number of articles: %d", len(articles))
		}
	})

	t.Run("Update device last seen", func(t *testing.T) {
		res, err := UpdateLastSeenForDevice(db, userDevice)
		if err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}

		if res != 1 {
			t.Fatalf("Expected 1, got %d", res)
		}

		updatedDevice, err := GetLegacyDevice(db, legacySyncCode, 1)
		if err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}

		if updatedDevice.LastSeen <= userDevice.LastSeen {
			t.Fatalf("New value %d is not greater than old value %d", updatedDevice.LastSeen, userDevice.LastSeen)
		}
	})

	t.Run("GetLegacyDevice fails no such device", func(t *testing.T) {
		_, err := GetLegacyDevice(db, legacySyncCode, 9999)
		if err == nil {
			t.Fatalf("Expected error")
		}

		if err != store.ErrNoSuchDevice {
			t.Fatalf("Expected ErrNoSuchDevice, not: %s", err.Error())
		}
	})

	t.Run("Feeds", func(t *testing.T) {
		// Initial get is empty
		feeds, err := GetLegacyFeeds(db, userDevice.UserId)
		if err == nil {
			t.Fatalf("Expected error on first query not %q", feeds)
		} else {
			if err != store.ErrNoFeeds {
				t.Fatalf("Unexpected error: %s", err.Error())
			}
		}

		// Add some feeds
		count, err := UpdateLegacyFeeds(db,
			userDevice.UserDbId,
			1,
			"content",
			"99",
		)
		if err != nil {
			t.Fatalf("error: %s", err.Error())
		}

		if count != 1 {
			t.Fatalf("Count is not 1: %d", count)
		}

		// New update comes in
		count, err = UpdateLegacyFeeds(db,
			userDevice.UserDbId,
			2,
			"content2",
			"101",
		)
		if err != nil {
			t.Fatalf("error: %s", err.Error())
		}

		if count != 1 {
			t.Fatalf("Count is not 1: %d", count)
		}

		// Now get the value
		feeds, err = GetLegacyFeeds(db, userDevice.UserId)
		if err != nil {
			t.Fatalf("Error: %s", err.Error())
		}

		if feeds.ContentHash != 2 {
			t.Errorf("Incorrect contenthash: %d", feeds.ContentHash)
		}
		if feeds.Content != "content2" {
			t.Errorf("Incorrect content: %s", feeds.Content)
		}
		if feeds.Etag != "101" {
			t.Errorf("Incorrect etag: %s", feeds.Etag)
		}
	})
}
