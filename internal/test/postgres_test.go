package test

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/spacecowboy/feeder-sync/internal/store"
	postgresqlStore "github.com/spacecowboy/feeder-sync/internal/store/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

const (
	user     = "username"
	password = "password"
	dbname   = "feedertest"
)

type PostgresContainerTestSuite struct {
	suite.Suite
	pgContainer *postgres.PostgresContainer
	ctx         context.Context
	snapshotDb  *sql.DB
}

func (suite *PostgresContainerTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	suite.pgContainer = NewContainer(suite.T(), suite.ctx)
	suite.snapshotDb = NewDb(suite.T(), suite.ctx, suite.pgContainer)
}

func (suite *PostgresContainerTestSuite) TearDownSuite() {
	suite.snapshotDb.Close()
	if err := suite.pgContainer.Terminate(suite.ctx); err != nil {
		suite.T().Fatal(err)
	}
}

func TestPostgresStore(t *testing.T) {
	ctx := context.Background()
	container := NewContainer(t, ctx)

	snapShotDp := NewDb(t, ctx, container)
	defer snapShotDp.Close()

	t.Log(snapShotDp.Stats())

	t.Run("Register new user works", func(t *testing.T) {
		db := postgresqlStore.PostgresStore{
			Db: NewDb(t, ctx, container),
		}
		defer db.Close()

		userDevice, err := db.RegisterNewUser(ctx, "devicename")

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

		devices, err := db.GetDevices(ctx, userDevice.UserId)

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

	t.Run("AddDeviceToChainWithLegacy no such user fails", func(t *testing.T) {
		db := postgresqlStore.PostgresStore{
			Db: NewDb(t, ctx, container),
		}
		defer db.Close()

		_, err := db.AddDeviceToChainWithLegacy(ctx, "foo bar", "bla bla")
		if err == nil {
			t.Fatalf("Expected a failure")
		}

		if got := err.Error(); got != "user not found" {
			t.Fatalf("error should be %q, not %q", "user not found", got)
		}
	})

	t.Run("AddDeviceToChainWithLegacy succeeds", func(t *testing.T) {
		db := postgresqlStore.PostgresStore{
			Db: NewDb(t, ctx, container),
		}
		defer db.Close()

		userDevice, err := db.RegisterNewUser(ctx, "firstDevice")
		if err != nil {
			t.Fatalf("Failed: %s", err.Error())
		}

		device, err := db.AddDeviceToChainWithLegacy(ctx, userDevice.LegacySyncCode, "secondDevice")
		if err != nil {
			t.Fatalf("failed: %s", err.Error())
		}

		if got := device.DeviceName; got != "secondDevice" {
			t.Errorf("Wrong device name: %s", got)
		}
	})

	t.Run("AddDeviceToChain no such user fails", func(t *testing.T) {
		db := postgresqlStore.PostgresStore{
			Db: NewDb(t, ctx, container),
		}
		defer db.Close()

		_, err := db.AddDeviceToChain(ctx, uuid.New(), "bla bla")
		if err == nil {
			t.Fatalf("Expected a failure")
		}

		if got := err.Error(); got != "user not found" {
			t.Fatalf("error should be %q, not %q", "user not found", got)
		}
	})

	t.Run("AddDeviceToChain succeeds", func(t *testing.T) {
		db := postgresqlStore.PostgresStore{
			Db: NewDb(t, ctx, container),
		}
		defer db.Close()

		userDevice, err := db.RegisterNewUser(ctx, "firstDevice")
		if err != nil {
			t.Fatalf("Failed: %s", err.Error())
		}

		device, err := db.AddDeviceToChain(ctx, userDevice.UserId, "otherDevice")
		if err != nil {
			t.Fatalf("failed: %s", err.Error())
		}

		if got := device.DeviceName; got != "otherDevice" {
			t.Errorf("Wrong device name: %s", got)
		}
	})

	t.Run("Migration invalid synccode returns error", func(t *testing.T) {
		db := postgresqlStore.PostgresStore{
			Db: NewDb(t, ctx, container),
		}
		defer db.Close()

		_, err := db.EnsureMigration(ctx, "tooshort", 1, "foo")

		if err == nil {
			t.Error("Expected an error")
		}
	})

	t.Run("Ensure migration works", func(t *testing.T) {
		db := postgresqlStore.PostgresStore{
			Db: NewDb(t, ctx, container),
		}
		defer db.Close()

		legacySyncCode := "fa18973dd5889b64d8ec2a08ede95d94ee07d430d0d1b80b11bfd6a0375552c0"
		_, err := db.EnsureMigration(ctx, legacySyncCode, 1, "devicename")
		if err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}

		_, err = db.GetLegacyDevice(ctx, legacySyncCode, 1)
		if err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}

		var wantRows int64

		legacySyncCode = "ba18973dd5889b64d8ec2a08ede95d94ee07d430d0d1b80b11bfd6a0375552c0"
		wantRows = 2
		got, err := db.EnsureMigration(ctx, legacySyncCode, 66, "devicename")
		if err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}
		if wantRows != got {
			t.Errorf("Wanted %d rows, but was %d", wantRows, got)
		}

		// Add another device
		wantRows = 1
		got, err = db.EnsureMigration(ctx, legacySyncCode, 67, "devicename")
		if err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}
		if wantRows != got {
			t.Errorf("Wanted %d rows, but was %d", wantRows, got)
		}

		// Same device again
		wantRows = 0
		got, err = db.EnsureMigration(ctx, legacySyncCode, 67, "devicename")
		if err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}
		if wantRows != got {
			t.Errorf("Wanted %d rows, but was %d", wantRows, got)
		}

		// Ensure data is correct
		rows, err := db.Db.Query(
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
		db := postgresqlStore.PostgresStore{
			Db: NewDb(t, ctx, container),
		}
		defer db.Close()

		legacySyncCode := "fa18973dd5889b64d8ec2a08ede95d94ee07d430d0d1b80b11bfd6a0375552c0"
		_, err := db.EnsureMigration(ctx, legacySyncCode, 1, "devicename")
		if err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}

		userDevice, err := db.GetLegacyDevice(ctx, legacySyncCode, 1)
		if err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}

		articles, err := db.GetArticles(ctx, userDevice.UserId, 0)
		if err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}

		if len(articles) != 0 {
			t.Fatalf("Expected no articles yet: %d", len(articles))
		}

		if err = db.AddLegacyArticle(ctx, userDevice.UserDbId, "first"); err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}

		// Now should get one
		articles, err = db.GetArticles(ctx, userDevice.UserId, 0)
		if err != nil {
			t.Fatalf("Got an error:%s", err.Error())
		}

		if len(articles) != 1 {
			t.Fatalf("Wrong number of articles: %d", len(articles))
		}

		article := articles[0]
		articles, err = db.GetArticles(ctx, userDevice.UserId, article.UpdatedAt)
		if err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}

		if len(articles) != 0 {
			t.Fatalf("Wrong number of articles: %d", len(articles))
		}
	})

	t.Run("Update device last seen", func(t *testing.T) {
		db := postgresqlStore.PostgresStore{
			Db: NewDb(t, ctx, container),
		}
		defer db.Close()

		legacySyncCode := "fa18973dd5889b64d8ec2a08ede95d94ee07d430d0d1b80b11bfd6a0375552c0"
		_, err := db.EnsureMigration(ctx, legacySyncCode, 1, "devicename")
		if err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}

		userDevice, err := db.GetLegacyDevice(ctx, legacySyncCode, 1)
		if err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}

		etag1, err := db.GetLegacyDevicesEtag(ctx, legacySyncCode)
		if err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}
		assert.NotEqual(t, "", etag1, "Etag should not be empty")

		res, err := db.UpdateLastSeenForDevice(ctx, userDevice)
		if err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}

		if res != 1 {
			t.Fatalf("Expected 1, got %d", res)
		}

		updatedDevice, err := db.GetLegacyDevice(ctx, legacySyncCode, 1)
		if err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}

		if updatedDevice.LastSeen <= userDevice.LastSeen {
			t.Fatalf("New value %d is not greater than old value %d", updatedDevice.LastSeen, userDevice.LastSeen)
		}

		etag2, err := db.GetLegacyDevicesEtag(ctx, legacySyncCode)
		if err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}

		assert.Equal(t, etag1, etag2, "Etag should not depend on lastSeen")
	})

	t.Run("GetLegacyDevice fails no such device", func(t *testing.T) {
		db := postgresqlStore.PostgresStore{
			Db: NewDb(t, ctx, container),
		}
		defer db.Close()

		legacySyncCode := "fa18973dd5889b64d8ec2a08ede95d94ee07d430d0d1b80b11bfd6a0375552c0"
		_, err := db.EnsureMigration(ctx, legacySyncCode, 1, "devicename")
		if err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}
		_, err = db.GetLegacyDevice(ctx, legacySyncCode, 9999)
		if err == nil {
			t.Fatalf("Expected error")
		}

		if err != store.ErrNoSuchDevice {
			t.Fatalf("Expected ErrNoSuchDevice, not: %s", err.Error())
		}
	})

	t.Run("GetLegacyDevicesEtag fails no such device", func(t *testing.T) {
		db := postgresqlStore.PostgresStore{
			Db: NewDb(t, ctx, container),
		}
		defer db.Close()

		legacySyncCode := "fa18973dd5889b64d8ec2a08ede95d94ee07d430d0d1b80b11bfd6a0375552c0"

		etag, err := db.GetLegacyDevicesEtag(ctx, legacySyncCode)
		assert.Equal(t, "", etag)
		assert.Equal(t, store.ErrNoSuchDevice, err)
	})

	t.Run("Feeds", func(t *testing.T) {
		db := postgresqlStore.PostgresStore{
			Db: NewDb(t, ctx, container),
		}
		defer db.Close()

		legacySyncCode := "fa18973dd5889b64d8ec2a08ede95d94ee07d430d0d1b80b11bfd6a0375552c0"
		_, err := db.EnsureMigration(ctx, legacySyncCode, 1, "devicename")
		if err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}

		userDevice, err := db.GetLegacyDevice(ctx, legacySyncCode, 1)
		if err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}

		// Initial get is empty
		feeds, err := db.GetLegacyFeeds(ctx, userDevice.UserId)
		if err == nil {
			t.Fatalf("Expected error on first query not %q", feeds)
		} else {
			if err != store.ErrNoFeeds {
				t.Fatalf("Unexpected error: %s", err.Error())
			}
		}

		// Add some feeds
		count, err := db.UpdateLegacyFeeds(
			ctx,
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

		etag, err := db.GetLegacyFeedsEtag(ctx, userDevice.UserId)
		if err != nil {
			t.Fatalf("Error: %s", err.Error())
		}
		assert.Equal(t, etag, "99")

		// New update comes in
		count, err = db.UpdateLegacyFeeds(
			ctx,
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
		feeds, err = db.GetLegacyFeeds(ctx, userDevice.UserId)
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
