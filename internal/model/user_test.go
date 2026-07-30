package model

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"testing"

	"invest/internal/store"
)

const lastInsertIDFailDriverName = "last_insert_id_fail"

func init() {
	sql.Register(lastInsertIDFailDriverName, lastInsertIDFailDriver{})
}

// openTestDB returns a SQLite DB migrated through the same path as production.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := store.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db.DB
}

// TestSeedAdminPropagatesCountError verifies that a failure of the COUNT(*)
// query is surfaced to the caller (Requirement 9.3) rather than silently
// proceeding under the "no users exist" assumption.
func TestSeedAdminPropagatesCountError(t *testing.T) {
	db := openTestDB(t)
	users := NewUserStore(db)

	// Drop the users table so the COUNT query fails.
	if _, err := db.Exec(`DROP TABLE users`); err != nil {
		t.Fatalf("drop table: %v", err)
	}
	err := users.SeedAdmin("admin", "strongpass")
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
	users := NewUserStore(db)
	if err := users.SeedAdmin("admin", "strongpass"); err != nil {
		t.Fatalf("seed admin: %v", err)
	}
	u, err := users.GetByUsername("admin")
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
	users := NewUserStore(db)
	if _, err := users.Create("alice", "strongpass", "user"); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := users.SeedAdmin("admin", "strongpass"); err != nil {
		t.Fatalf("seed admin should be a no-op when users exist: %v", err)
	}
	if _, err := users.GetByUsername("admin"); err == nil {
		t.Error("admin should not have been created")
	}
}

func TestUserStoreRejectsInvalidRoleAndStatus(t *testing.T) {
	db := openTestDB(t)
	users := NewUserStore(db)

	if _, err := users.Create("alice", "strongpass", "owner"); !errors.Is(err, ErrInvalidRole) {
		t.Fatalf("Create invalid role err = %v, want ErrInvalidRole", err)
	}

	user, err := users.Create("alice", "strongpass", RoleUser)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := users.Update(user.ID, "owner", StatusEnabled); !errors.Is(err, ErrInvalidRole) {
		t.Fatalf("Update invalid role err = %v, want ErrInvalidRole", err)
	}
	if err := users.Update(user.ID, RoleAdmin, 3); !errors.Is(err, ErrInvalidStatus) {
		t.Fatalf("Update invalid status err = %v, want ErrInvalidStatus", err)
	}
}

func TestUserStoreUpdateDeleteMissingUser(t *testing.T) {
	db := openTestDB(t)
	users := NewUserStore(db)

	if _, err := users.GetByUsername("missing"); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("GetByUsername missing user err = %v, want ErrUserNotFound", err)
	}
	if _, err := users.GetByID(404); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("GetByID missing user err = %v, want ErrUserNotFound", err)
	}
	if err := users.Update(404, RoleUser, StatusEnabled); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("Update missing user err = %v, want ErrUserNotFound", err)
	}
	if err := users.Delete(404); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("Delete missing user err = %v, want ErrUserNotFound", err)
	}
}

func TestUserStoreCreatePropagatesLastInsertIDError(t *testing.T) {
	db, err := sql.Open(lastInsertIDFailDriverName, "")
	if err != nil {
		t.Fatalf("open fake db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	users := NewUserStore(db)
	_, err = users.Create("alice", "strongpass", RoleUser)
	if err == nil {
		t.Fatal("expected LastInsertId error")
	}
	if !strings.Contains(err.Error(), "insert user id") {
		t.Fatalf("error should include LastInsertId context, got %v", err)
	}
}

type lastInsertIDFailDriver struct{}

func (lastInsertIDFailDriver) Open(string) (driver.Conn, error) {
	return lastInsertIDFailConn{}, nil
}

type lastInsertIDFailConn struct{}

func (lastInsertIDFailConn) Prepare(string) (driver.Stmt, error) {
	return nil, fmt.Errorf("unsupported prepare")
}
func (lastInsertIDFailConn) Close() error              { return nil }
func (lastInsertIDFailConn) Begin() (driver.Tx, error) { return nil, fmt.Errorf("unsupported tx") }

func (lastInsertIDFailConn) ExecContext(context.Context, string, []driver.NamedValue) (driver.Result, error) {
	return lastInsertIDFailResult{}, nil
}

type lastInsertIDFailResult struct{}

func (lastInsertIDFailResult) LastInsertId() (int64, error) {
	return 0, fmt.Errorf("last insert id unavailable")
}
func (lastInsertIDFailResult) RowsAffected() (int64, error) { return 1, nil }
