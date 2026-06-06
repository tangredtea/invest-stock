package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"invest/internal/auth"
	"invest/internal/model"
	"invest/internal/store"
)

// newTestAuthApp builds an authApp backed by a temporary SQLite database.
func newTestAuthApp(t *testing.T) *authApp {
	t.Helper()
	dbPath := t.TempDir() + "/test.db"
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	users := model.NewUserStore(db.DB)
	return &authApp{
		jwt:     auth.NewJWTManager(strings.Repeat("k", 40), time.Hour),
		users:   users,
		limiter: NewLoginRateLimiter(3, defaultLoginWindow),
	}
}

func TestHandleLoginRateLimited(t *testing.T) {
	// Requirements 4.1, 4.3: after reaching the failure threshold, further
	// attempts are rejected with 429 without credential validation.
	app := newTestAuthApp(t)

	body := `{"username":"nobody","password":"wrongpass"}`
	doLogin := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
		req.RemoteAddr = "203.0.113.5:12345"
		rec := httptest.NewRecorder()
		app.handleLogin(rec, req)
		return rec
	}

	// 3 failed attempts (threshold = 3) → each returns 401.
	for i := 0; i < 3; i++ {
		if rec := doLogin(); rec.Code != 401 {
			t.Fatalf("attempt %d: expected 401, got %d", i+1, rec.Code)
		}
	}
	// 4th attempt is now rate-limited.
	if rec := doLogin(); rec.Code != 429 {
		t.Fatalf("expected 429 after threshold reached, got %d", rec.Code)
	}
}

func TestHandleLoginResetsOnSuccess(t *testing.T) {
	// Requirement 4.5: a successful login clears the failure count for the IP.
	app := newTestAuthApp(t)
	if _, err := app.users.Create("alice", "strongpass", "user"); err != nil {
		t.Fatalf("create user: %v", err)
	}

	ip := "203.0.113.9:54321"
	post := func(b string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(b))
		req.RemoteAddr = ip
		rec := httptest.NewRecorder()
		app.handleLogin(rec, req)
		return rec
	}

	// Two failures (below threshold of 3).
	post(`{"username":"alice","password":"wrong"}`)
	post(`{"username":"alice","password":"wrong"}`)
	// A successful login resets the counter.
	if rec := post(`{"username":"alice","password":"strongpass"}`); rec.Code != 200 {
		t.Fatalf("expected 200 on valid login, got %d", rec.Code)
	}
	// After reset, several more failures should still be allowed (not yet limited).
	for i := 0; i < 2; i++ {
		if rec := post(`{"username":"alice","password":"wrong"}`); rec.Code != 401 {
			t.Fatalf("post-reset failure %d: expected 401, got %d", i+1, rec.Code)
		}
	}
}


// authedRequest builds a request with claims attached as if requireAuth had
// already approved it, so the per-handler self-account checks can be unit-tested.
func authedRequest(method, target string, body string, uid int64, role string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	claims := &auth.Claims{UserID: uid, Username: "admin", Role: role}
	ctx := context.WithValue(req.Context(), claimsKey, claims)
	return req.WithContext(ctx)
}

func TestHandleUpdateUserBlocksSelf(t *testing.T) {
	// Requirement 9.1: an admin cannot disable/modify their own account.
	app := newTestAuthApp(t)
	u, err := app.users.Create("admin", "strongpass", "admin")
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}
	body := `{"role":"admin","status":0}`
	req := authedRequest(http.MethodPut, "/api/admin/users/"+itoa(u.ID), body, u.ID, "admin")
	rec := httptest.NewRecorder()
	app.handleUpdateUser(rec, req)
	if rec.Code != 400 {
		t.Errorf("expected 400 when admin updates self, got %d", rec.Code)
	}
	// Status must still be 1 (record unchanged).
	got, _ := app.users.GetByID(u.ID)
	if got.Status != 1 {
		t.Errorf("status should remain 1, got %d", got.Status)
	}
}

func TestHandleDeleteUserBlocksSelf(t *testing.T) {
	// Requirement 9.2: an admin cannot delete their own account.
	app := newTestAuthApp(t)
	u, err := app.users.Create("admin", "strongpass", "admin")
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}
	req := authedRequest(http.MethodDelete, "/api/admin/users/"+itoa(u.ID), "", u.ID, "admin")
	rec := httptest.NewRecorder()
	app.handleDeleteUser(rec, req)
	if rec.Code != 400 {
		t.Errorf("expected 400 when admin deletes self, got %d", rec.Code)
	}
	// Record must still exist.
	if _, err := app.users.GetByID(u.ID); err != nil {
		t.Errorf("admin record should still exist: %v", err)
	}
}

func TestHandleDeleteUserAllowsOthers(t *testing.T) {
	app := newTestAuthApp(t)
	admin, _ := app.users.Create("admin", "strongpass", "admin")
	target, _ := app.users.Create("bob", "strongpass", "user")

	req := authedRequest(http.MethodDelete, "/api/admin/users/"+itoa(target.ID), "", admin.ID, "admin")
	rec := httptest.NewRecorder()
	app.handleDeleteUser(rec, req)
	if rec.Code != 200 {
		t.Errorf("expected 200 when admin deletes another user, got %d", rec.Code)
	}
}

// itoa is a tiny local helper to avoid pulling strconv into the test header.
func itoa(n int64) string {
	return strconv.FormatInt(n, 10)
}
