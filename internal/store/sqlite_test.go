package store

import (
	"database/sql"
	"testing"
)

func TestOpenCreatesParentDirectoryAndMigratesUsersTable(t *testing.T) {
	db, err := Open(t.TempDir() + "/nested/invest.db")
	if err != nil {
		t.Fatalf("Open error: %v", err)
	}
	defer db.Close()

	columns := userColumns(t, db.DB)
	for _, name := range []string{"id", "username", "password", "role", "status", "created_at", "updated_at"} {
		if !columns[name] {
			t.Fatalf("users table missing column %q; columns=%v", name, columns)
		}
	}
}

func TestOpenEnablesWALMode(t *testing.T) {
	db, err := Open(t.TempDir() + "/invest.db")
	if err != nil {
		t.Fatalf("Open error: %v", err)
	}
	defer db.Close()

	var mode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatalf("read journal_mode: %v", err)
	}
	if mode != "wal" {
		t.Fatalf("journal_mode = %q, want wal", mode)
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	db, err := sql.Open("sqlite", t.TempDir()+"/invest.db")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	if err := migrate(db); err != nil {
		t.Fatalf("first migrate: %v", err)
	}
	if err := migrate(db); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
}

func userColumns(t *testing.T, db *sql.DB) map[string]bool {
	t.Helper()
	rows, err := db.Query("PRAGMA table_info(users)")
	if err != nil {
		t.Fatalf("table_info users: %v", err)
	}
	defer rows.Close()

	columns := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			t.Fatalf("scan column: %v", err)
		}
		columns[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate columns: %v", err)
	}
	return columns
}
