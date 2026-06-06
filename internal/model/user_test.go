package model

import (
	"database/sql"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

// openTestDB returns an in-memory SQLite DB with the users table created.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(`
		CREATE TABLE users (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			username   TEXT    NOT NULL UNIQUE,
			password   TEXT    NOT NULL,
			role       TEXT    NOT NULL DEFAULT 'user',
			status     INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		t.Fatalf("create users table: %v", err)
	}
	return db
}

// TestSeedAdminPropagatesCountError verifies that a failure of the COUNT(*)
// query is surfaced to the caller (Requirement 9.3) rather than silently
// proceeding under the "no users exist" assumption.
func TestSeedAdminPropagatesCountError(t *testing.T) {
	db := openTestDB(t)
	store := NewUserStore(db)

	// Drop the users table so the COUNT query fails.
	if _, err := db.Exec(`DROP TABLE users`); err != nil {
		t.Fatalf("drop table: %v", err)
	}
	err := store.SeedAdmin("admin", "strongpass")
	if err == nil {
		t.Fatal("expected error when COUNT query fails, got nil")
	}
	if !strings.Contains(err.Error(), "seed admin count") {
		t.Errorf("error should be wrapped with context, got: %v", err)
	}
}

// TestSeedAdminCreatesAdminWhenEmpty verifies the success path: an admin is
// created when no users exist.
func TestSeedAdminCreatesAdminWhenEmpty(t *testing.T) {
	db := openTestDB(t)
	store := NewUserStore(db)
	if err := store.SeedAdmin("admin", "strongpass"); err != nil {
		t.Fatalf("seed admin: %v", err)
	}
	u, err := store.GetByUsername("admin")
	if err != nil {
		t.Fatalf("get admin: %v", err)
	}
	if u.Role != "admin" {
		t.Errorf("expected role=admin, got %q", u.Role)
	}
}

// TestSeedAdminSkipsWhenUsersExist verifies the no-op path.
func TestSeedAdminSkipsWhenUsersExist(t *testing.T) {
	db := openTestDB(t)
	store := NewUserStore(db)
	if _, err := store.Create("alice", "strongpass", "user"); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := store.SeedAdmin("admin", "strongpass"); err != nil {
		t.Fatalf("seed admin should be a no-op when users exist: %v", err)
	}
	if _, err := store.GetByUsername("admin"); err == nil {
		t.Error("admin should not have been created")
	}
}
