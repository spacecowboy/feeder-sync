package store_transfer

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/migrators/golangmigrator"
	postgresstore "github.com/spacecowboy/feeder-sync/internal/store/postgres"
	sqlitestore "github.com/spacecowboy/feeder-sync/internal/store/sqlite"

	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Move common code to shared utils instead

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
	return func(req *testcontainers.GenericContainerRequest) error {
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
		return nil
	}
}

func NewStore(t *testing.T, ctx context.Context, container *postgres.PostgresContainer) postgresstore.PostgresStore {
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
	return postgresstore.PostgresStore{
		Db: pgtestdb.New(t, conf, migrator),
	}
}

func NewSqliteStore(t *testing.T) sqlitestore.SqliteStore {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "sqlite.db")
	t.Logf("DB path : %s\n", dbPath)

	sqliteStore, err := sqlitestore.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %s", err.Error())
	}

	err = sqliteStore.RunMigrations("file://../../../migrations_sqlite")
	if err != nil {
		t.Fatalf("Migration failed: %s", err.Error())
	}

	return sqliteStore
}

func TestMovingFromSqliteToPostgres(t *testing.T) {
	ctx := context.Background()
	container, err := NewContainer(t, ctx)

	if err != nil {
		t.Fatalf("Failed to start postgres container: %s", err.Error())
	}

	t.Run("move_from_sqlite_to_postgres", func(t *testing.T) {
		psqlStore := NewStore(t, ctx, container)
		defer psqlStore.Close()

		sqliteStore := NewSqliteStore(t)
		defer sqliteStore.Close()

		// Add some data to sqlite
		user1, err := sqliteStore.RegisterNewUser("device1")
		if err != nil {
			t.Fatalf("Failed to register new user: %s", err.Error())
		}

		rows, err := sqliteStore.UpdateLegacyFeeds(
			user1.UserDbId,
			123,
			"feeds1",
			"etag1",
		)
		if err != nil {
			t.Fatalf("Failed to update legacy feeds: %s", err.Error())
		}

		assert.Equal(t, rows, int64(1))

		err = sqliteStore.AddLegacyArticle(
			user1.UserDbId,
			"article1",
		)
		if err != nil {
			t.Fatalf("Failed to add legacy article: %s", err.Error())
		}

		// Move data from sqlite to postgres
		err = MoveBetweenStores(&sqliteStore, &psqlStore)
		if err != nil {
			t.Fatalf("Failed to move data: %s", err.Error())
		}

		// Verify data in postgres
		toDevices, err := psqlStore.GetDevices(user1.UserId)
		if err != nil {
			t.Fatalf("Failed to get devices: %s", err.Error())
		}

		assert.Len(t, toDevices, 1)

		toFeeds, err := psqlStore.GetLegacyFeeds(user1.UserId)
		if err != nil {
			t.Fatalf("Failed to get legacy feeds: %s", err.Error())
		}

		assert.Equal(t, toFeeds.ContentHash, int64(123))
	})
}
