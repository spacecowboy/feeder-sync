package sqlite

import (
	"path/filepath"
	"testing"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

func TestStoreApi(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "sqlite.db")
	t.Logf("DB path : %s\n", dbPath)

	store, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %q", err)
	}

	defer func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Failed to close store: %q", err)
			return
		}
	}()

	err = store.RunMigrations("file://../../../migrations")
	if err != nil {
		t.Fatalf("Migration failed: %q", err)
	}

	t.Run("Invalid synccode returns error", func(t *testing.T) {
		_, err := store.EnsureMigration("tooshort", 1, "foo")

		if err == nil {
			t.Error("Expected an error")
		}
	})

	t.Run("Ensure migration works", func(t *testing.T) {
		var wantRows int64

		legacySyncCode := "ba18973dd5889b64d8ec2a08ede95d94ee07d430d0d1b80b11bfd6a0375552c0"
		wantRows = 2
		got, err := store.EnsureMigration(legacySyncCode, 1, "devicename")
		if err != nil {
			t.Fatalf("Got an error: %q", err)
		}
		if wantRows != got {
			t.Errorf("Wanted %d rows, but was %d", wantRows, got)
		}

		// Add another device
		wantRows = 1
		got, err = store.EnsureMigration(legacySyncCode, 2, "devicename")
		if err != nil {
			t.Fatalf("Got an error: %q", err)
		}
		if wantRows != got {
			t.Errorf("Wanted %d rows, but was %d", wantRows, got)
		}

		// Same device again
		wantRows = 0
		got, err = store.EnsureMigration(legacySyncCode, 2, "devicename")
		if err != nil {
			t.Fatalf("Got an error: %q", err)
		}
		if wantRows != got {
			t.Errorf("Wanted %d rows, but was %d", wantRows, got)
		}

		// Ensure data is correct
		rows, err := store.db.Query("select device_id, legacy_device_id, device_name, last_seen, user_db_id from devices")
		if err != nil {
			t.Fatalf("Got an error: %q", err)
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
				t.Fatalf("Got an error: %q", err)
			}

			if userDbId != 1 {
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
			t.Fatalf("Got an error: %q", err)
		}

		if deviceCount != 2 {
			t.Errorf("Wanted 2, but device count: %d", deviceCount)
		}
	})
}

func TestMigrations(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "sqlite.db")
	t.Logf("DB path : %s\n", dbPath)

	t.Run("Run migrations", func(t *testing.T) {
		store, err := New(dbPath)
		if err != nil {
			t.Fatalf("Failed to create store: %q", err)
		}

		defer func() {
			if err := store.Close(); err != nil {
				t.Fatalf("Failed to close store: %q", err)
				return
			}
		}()

		err = store.RunMigrations("file://../../../migrations")
		if err != nil {
			t.Errorf("Migration failed: %q", err)
		}
	})
}
