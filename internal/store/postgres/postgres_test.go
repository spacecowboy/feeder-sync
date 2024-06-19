package postgres

import (
	"context"
	"testing"
	"time"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/migrators/golangmigrator"
	"github.com/spacecowboy/feeder-sync/internal/store"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	user     = "username"
	password = "password"
	dbname   = "feedertest"
)

// NewDB is a helper that returns an open connection to a unique and isolated
// test database, fully migrated and ready for you to query.
func NewContainer(t *testing.T, ctx context.Context) (*postgres.PostgresContainer, error) {
	t.Helper()

	container, err := postgres.RunContainer(
		ctx,
		testcontainers.WithImage("postgres:15"),
		postgres.WithDatabase(dbname),
		postgres.WithUsername(user),
		postgres.WithPassword(password),
		WithTmpfs(),
		// postgres.WithInitScripts(
		// 	"../../../migrations_postgres/1_create_tables.up.sql",
		// 	"../../../migrations_postgres/2_create_articles.up.sql",
		// 	"../../../migrations_postgres/3_add_updated_at.up.sql",
		// 	"../../../migrations_postgres/4_create_legacy_feeds.up.sql",
		// 	"../../../migrations_postgres/5_add_updated_at_index.up.sql",
		// ),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(5*time.Second),
		),
	)

	if err != nil {
		t.Fatalf("Failed to start postgres container: %s", err.Error())
	}

	// Clean up the container after the test is complete
	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %s", err)
		}
	})

	return container, nil
}

func WithTmpfs() testcontainers.CustomizeRequestOption {
	return func(req *testcontainers.GenericContainerRequest) {
		req.Tmpfs = map[string]string{"/var/lib/postgresql/data": "rw"}
		req.Env["PGDATA"] = "/var/lib/postgresql/data"
		req.Cmd = []string{
			"postgres",
			"-c",
			// turn off fsync for speed
			"fsync=off",
			"-c",
			// log everything for debugging
			"log_statement=all",
		}
	}
}

func NewStore(t *testing.T, ctx context.Context, container *postgres.PostgresContainer) PostgresStore {
	t.Helper()

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("Failed to get host: %s", err.Error())
	}

	port, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		t.Fatalf("Failed to get port: %s", err.Error())
	}

	conf := pgtestdb.Config{
		DriverName: "postgres",
		User:       user,
		Password:   password,
		Database:   dbname,
		Host:       host,
		Port:       port.Port(),
		Options:    "sslmode=disable",
	}

	migrator := golangmigrator.New("../../../migrations_postgres")
	return PostgresStore{
		Db: pgtestdb.New(t, conf, migrator),
	}
}

func TestPostgresIntegration(t *testing.T) {
	ctx := context.Background()
	container, err := NewContainer(t, ctx)
	if err != nil {
		t.Fatalf("Failed to start postgres container: %s", err.Error())
	}

	t.Parallel()

	t.Run("Register new user works", func(t *testing.T) {
		db := NewStore(t, ctx, container)
		defer db.Close()

		userDevice, err := db.RegisterNewUser("devicename")

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

		devices, err := db.GetDevices(userDevice.UserId)

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
		db := NewStore(t, ctx, container)
		defer db.Close()

		_, err = db.AddDeviceToChainWithLegacy("foo bar", "bla bla")
		if err == nil {
			t.Fatalf("Expected a failure")
		}

		if got := err.Error(); got != "user not found" {
			t.Fatalf("error should be %q, not %q", "user not found", got)
		}
	})

	t.Run("AddDeviceToChainWithLegacy succeeds", func(t *testing.T) {
		db := NewStore(t, ctx, container)
		defer db.Close()

		userDevice, err := db.RegisterNewUser("firstDevice")
		if err != nil {
			t.Fatalf("Failed: %s", err.Error())
		}

		device, err := db.AddDeviceToChainWithLegacy(userDevice.LegacySyncCode, "secondDevice")
		if err != nil {
			t.Fatalf("failed: %s", err.Error())
		}

		if got := device.DeviceName; got != "secondDevice" {
			t.Errorf("Wrong device name: %s", got)
		}
	})

	t.Run("AddDeviceToChain no such user fails", func(t *testing.T) {
		db := NewStore(t, ctx, container)
		defer db.Close()

		_, err = db.AddDeviceToChain(uuid.New(), "bla bla")
		if err == nil {
			t.Fatalf("Expected a failure")
		}

		if got := err.Error(); got != "user not found" {
			t.Fatalf("error should be %q, not %q", "user not found", got)
		}
	})

	t.Run("AddDeviceToChain succeeds", func(t *testing.T) {
		db := NewStore(t, ctx, container)
		defer db.Close()

		userDevice, err := db.RegisterNewUser("firstDevice")
		if err != nil {
			t.Fatalf("Failed: %s", err.Error())
		}

		device, err := db.AddDeviceToChain(userDevice.UserId, "otherDevice")
		if err != nil {
			t.Fatalf("failed: %s", err.Error())
		}

		if got := device.DeviceName; got != "otherDevice" {
			t.Errorf("Wrong device name: %s", got)
		}
	})

	t.Run("Migration invalid synccode returns error", func(t *testing.T) {
		db := NewStore(t, ctx, container)
		defer db.Close()

		_, err = db.EnsureMigration("tooshort", 1, "foo")

		if err == nil {
			t.Error("Expected an error")
		}
	})

	t.Run("Ensure migration works", func(t *testing.T) {
		db := NewStore(t, ctx, container)
		defer db.Close()

		legacySyncCode := "fa18973dd5889b64d8ec2a08ede95d94ee07d430d0d1b80b11bfd6a0375552c0"
		_, err = db.EnsureMigration(legacySyncCode, 1, "devicename")
		if err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}

		_, err := db.GetLegacyDevice(legacySyncCode, 1)
		if err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}

		var wantRows int64

		legacySyncCode = "ba18973dd5889b64d8ec2a08ede95d94ee07d430d0d1b80b11bfd6a0375552c0"
		wantRows = 2
		got, err := db.EnsureMigration(legacySyncCode, 66, "devicename")
		if err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}
		if wantRows != got {
			t.Errorf("Wanted %d rows, but was %d", wantRows, got)
		}

		// Add another device
		wantRows = 1
		got, err = db.EnsureMigration(legacySyncCode, 67, "devicename")
		if err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}
		if wantRows != got {
			t.Errorf("Wanted %d rows, but was %d", wantRows, got)
		}

		// Same device again
		wantRows = 0
		got, err = db.EnsureMigration(legacySyncCode, 67, "devicename")
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
		db := NewStore(t, ctx, container)
		defer db.Close()

		legacySyncCode := "fa18973dd5889b64d8ec2a08ede95d94ee07d430d0d1b80b11bfd6a0375552c0"
		_, err = db.EnsureMigration(legacySyncCode, 1, "devicename")
		if err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}

		userDevice, err := db.GetLegacyDevice(legacySyncCode, 1)
		if err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}

		articles, err := db.GetArticles(userDevice.UserId, 0)
		if err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}

		if len(articles) != 0 {
			t.Fatalf("Expected no articles yet: %d", len(articles))
		}

		if err = db.AddLegacyArticle(userDevice.UserDbId, "first"); err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}

		// Now should get one
		articles, err = db.GetArticles(userDevice.UserId, 0)
		if err != nil {
			t.Fatalf("Got an error:%s", err.Error())
		}

		if len(articles) != 1 {
			t.Fatalf("Wrong number of articles: %d", len(articles))
		}

		article := articles[0]
		articles, err = db.GetArticles(userDevice.UserId, article.UpdatedAt)
		if err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}

		if len(articles) != 0 {
			t.Fatalf("Wrong number of articles: %d", len(articles))
		}
	})

	t.Run("Update device last seen", func(t *testing.T) {
		db := NewStore(t, ctx, container)
		defer db.Close()

		legacySyncCode := "fa18973dd5889b64d8ec2a08ede95d94ee07d430d0d1b80b11bfd6a0375552c0"
		_, err = db.EnsureMigration(legacySyncCode, 1, "devicename")
		if err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}

		userDevice, err := db.GetLegacyDevice(legacySyncCode, 1)
		if err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}

		res, err := db.UpdateLastSeenForDevice(userDevice)
		if err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}

		if res != 1 {
			t.Fatalf("Expected 1, got %d", res)
		}

		updatedDevice, err := db.GetLegacyDevice(legacySyncCode, 1)
		if err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}

		if updatedDevice.LastSeen <= userDevice.LastSeen {
			t.Fatalf("New value %d is not greater than old value %d", updatedDevice.LastSeen, userDevice.LastSeen)
		}
	})

	t.Run("GetLegacyDevice fails no such device", func(t *testing.T) {
		db := NewStore(t, ctx, container)
		defer db.Close()

		legacySyncCode := "fa18973dd5889b64d8ec2a08ede95d94ee07d430d0d1b80b11bfd6a0375552c0"
		_, err = db.EnsureMigration(legacySyncCode, 1, "devicename")
		if err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}
		_, err = db.GetLegacyDevice(legacySyncCode, 9999)
		if err == nil {
			t.Fatalf("Expected error")
		}

		if err != store.ErrNoSuchDevice {
			t.Fatalf("Expected ErrNoSuchDevice, not: %s", err.Error())
		}
	})

	t.Run("Feeds", func(t *testing.T) {
		db := NewStore(t, ctx, container)
		defer db.Close()

		legacySyncCode := "fa18973dd5889b64d8ec2a08ede95d94ee07d430d0d1b80b11bfd6a0375552c0"
		_, err = db.EnsureMigration(legacySyncCode, 1, "devicename")
		if err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}

		userDevice, err := db.GetLegacyDevice(legacySyncCode, 1)
		if err != nil {
			t.Fatalf("Got an error: %s", err.Error())
		}

		// Initial get is empty
		feeds, err := db.GetLegacyFeeds(userDevice.UserId)
		if err == nil {
			t.Fatalf("Expected error on first query not %q", feeds)
		} else {
			if err != store.ErrNoFeeds {
				t.Fatalf("Unexpected error: %s", err.Error())
			}
		}

		// Add some feeds
		count, err := db.UpdateLegacyFeeds(
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
		count, err = db.UpdateLegacyFeeds(
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
		feeds, err = db.GetLegacyFeeds(userDevice.UserId)
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
