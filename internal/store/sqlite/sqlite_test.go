package sqlite

import (
	"path/filepath"
	"testing"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "modernc.org/sqlite"
)

// func TestStoreApi(t *testing.T) {
// 	dir := t.TempDir()
// 	dbPath := filepath.Join(dir, "sqlite.db")
// 	t.Logf("DB path : %s\n", dbPath)

// 	db, err := sql.Open("sqlite", dbPath)
// 	if err != nil {
// 		t.Fatalf("Can't create temp database: %q", err)
// 	}
// 	defer func() {
// 		if err := db.Close(); err != nil {
// 			return
// 		}
// 	}()

// 	t.Run("Ensure migration", func(t *testing.T) {
// 		store, err := New(dbPath)
// 		if err != nil {
// 			t.Fatalf("Failed to create store: %q", err)
// 		}
// 		t.Errorf("TODO")
// 	})
// }

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
