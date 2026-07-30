package main

import (
	"context"
	"net/http"
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
	app, _ := newTestAuthAppWithDB(t)
	return app
}

func newTestAuthAppWithDB(t *testing.T) (*authApp, *store.DB) {
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
	}, db
}

func createTestUser(t *testing.T, app *authApp, username, role string) *model.User {
	t.Helper()
	u, err := app.users.Create(username, "strongpass", role)
	if err != nil {
		t.Fatalf("create user %q: %v", username, err)
	}
	return u
}

func getTestUser(t *testing.T, app *authApp, id int64) *model.User {
	t.Helper()
	u, err := app.users.GetByID(id)
	if err != nil {
		t.Fatalf("get user %d: %v", id, err)
	}
	return u
}

func TestHandleLoginRateLimited(t *testing.T) {
	// Requirements 4.1, 4.3: after reaching the failure threshold, further
	// attempts are rejected with 429 without credential validation.
	app := newTestAuthApp(t)

	body := `{"username":"nobody","password":"wrongpass"}`
	doLogin := func() int {
		req := jsonTestRequest(http.MethodPost, "/api/auth/login", body)
		req.RemoteAddr = "203.0.113.5:12345"
		return recordTestRequest(req, app.handleLogin).Code
	}

	// 3 failed attempts (threshold = 3) → each returns 401.
	for i := 0; i < 3; i++ {
		if code := doLogin(); code != 401 {
			t.Fatalf("attempt %d: expected 401, got %d", i+1, code)
		}
	}
	// 4th attempt is now rate-limited.
	if code := doLogin(); code != 429 {
		t.Fatalf("expected 429 after threshold reached, got %d", code)
	}
}

func TestHandleLoginResetsOnSuccess(t *testing.T) {
	// Requirement 4.5: a successful login clears the failure count for the IP.
	app := newTestAuthApp(t)
	createTestUser(t, app, "alice", model.RoleUser)

	ip := "203.0.113.9:54321"
	post := func(b string) int {
		req := jsonTestRequest(http.MethodPost, "/api/auth/login", b)
		req.RemoteAddr = ip
		return recordTestRequest(req, app.handleLogin).Code
	}

	// Two failures (below threshold of 3).
	post(`{"username":"alice","password":"wrong"}`)
	post(`{"username":"alice","password":"wrong"}`)
	// A successful login resets the counter.
	if code := post(`{"username":"alice","password":"strongpass"}`); code != 200 {
		t.Fatalf("expected 200 on valid login, got %d", code)
	}
	// After reset, several more failures should still be allowed (not yet limited).
	for i := 0; i < 2; i++ {
		if code := post(`{"username":"alice","password":"wrong"}`); code != 401 {
			t.Fatalf("post-reset failure %d: expected 401, got %d", i+1, code)
		}
	}
}

func TestHandleRegisterRejectsDuplicateUsername(t *testing.T) {
	app := newTestAuthApp(t)
	createTestUser(t, app, "alice", model.RoleUser)

	body := `{"username":"alice","password":"strongpass"}`
	rec := postTestJSON("/api/auth/register", body, app.handleRegister)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for duplicate username, got %d", rec.Code)
	}

	var resp jsonResp
	decodeTestResponse(t, rec, &resp)
	if resp.Message != "用户名已存在" {
		t.Fatalf("message = %q, want %q", resp.Message, "用户名已存在")
	}
}

func TestHandleRegisterReturns500OnStoreError(t *testing.T) {
	app, db := newTestAuthAppWithDB(t)

	// Break the users table so Create fails with a non-duplicate DB error.
	if _, err := db.Exec(`DROP TABLE users`); err != nil {
		t.Fatalf("drop table: %v", err)
	}

	rec := postTestJSON("/api/auth/register", `{"username":"bob","password":"strongpass"}`, app.handleRegister)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for store failure, got %d", rec.Code)
	}

	var resp jsonResp
	decodeTestResponse(t, rec, &resp)
	if resp.Message != "注册失败" {
		t.Fatalf("message = %q, want %q", resp.Message, "注册失败")
	}
}

// authedRequest builds a request with claims attached as if requireAuth had
// already approved it, so the per-handler self-account checks can be unit-tested.
func authedRequest(method, target string, body string, uid int64, role string) *http.Request {
	req := jsonTestRequest(method, target, body)
	claims := &auth.Claims{UserID: uid, Username: "admin", Role: role}
	ctx := context.WithValue(req.Context(), claimsKey, claims)
	return req.WithContext(ctx)
}

func TestHandleUpdateUserBlocksSelf(t *testing.T) {
	// Requirement 9.1: an admin cannot disable/modify their own account.
	app := newTestAuthApp(t)
	u := createTestUser(t, app, "admin", model.RoleAdmin)
	body := `{"role":"admin","status":0}`
	req := authedRequest(http.MethodPut, "/api/admin/users/"+itoa(u.ID), body, u.ID, "admin")
	rec := recordTestRequest(req, app.handleUpdateUser)
	if rec.Code != 400 {
		t.Errorf("expected 400 when admin updates self, got %d", rec.Code)
	}
	// Status must still be 1 (record unchanged).
	got := getTestUser(t, app, u.ID)
	if got.Status != 1 {
		t.Errorf("status should remain 1, got %d", got.Status)
	}
}

func TestHandleDeleteUserBlocksSelf(t *testing.T) {
	// Requirement 9.2: an admin cannot delete their own account.
	app := newTestAuthApp(t)
	u := createTestUser(t, app, "admin", model.RoleAdmin)
	req := authedRequest(http.MethodDelete, "/api/admin/users/"+itoa(u.ID), "", u.ID, "admin")
	rec := recordTestRequest(req, app.handleDeleteUser)
	if rec.Code != 400 {
		t.Errorf("expected 400 when admin deletes self, got %d", rec.Code)
	}
	// Record must still exist.
	getTestUser(t, app, u.ID)
}

func TestHandleDeleteUserAllowsOthers(t *testing.T) {
	app := newTestAuthApp(t)
	admin := createTestUser(t, app, "admin", model.RoleAdmin)
	target := createTestUser(t, app, "bob", model.RoleUser)

	req := authedRequest(http.MethodDelete, "/api/admin/users/"+itoa(target.ID), "", admin.ID, "admin")
	rec := recordTestRequest(req, app.handleDeleteUser)
	if rec.Code != 200 {
		t.Errorf("expected 200 when admin deletes another user, got %d", rec.Code)
	}
}

func TestHandleUpdateUserRejectsInvalidRoleAndStatus(t *testing.T) {
	app := newTestAuthApp(t)
	admin := createTestUser(t, app, "admin", model.RoleAdmin)
	target := createTestUser(t, app, "bob", model.RoleUser)

	cases := []struct {
		name string
		body string
	}{
		{"invalid role", `{"role":"owner","status":1}`},
		{"invalid status", `{"role":"user","status":3}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := authedRequest(http.MethodPut, "/api/admin/users/"+itoa(target.ID), tc.body, admin.ID, "admin")
			rec := recordTestRequest(req, app.handleUpdateUser)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", rec.Code)
			}
		})
	}
}

func TestHandleUpdateDeleteMissingUserReturns404(t *testing.T) {
	app := newTestAuthApp(t)
	admin := createTestUser(t, app, "admin", model.RoleAdmin)

	update := authedRequest(http.MethodPut, "/api/admin/users/404", `{"role":"user","status":1}`, admin.ID, "admin")
	updateRec := recordTestRequest(update, app.handleUpdateUser)
	if updateRec.Code != http.StatusNotFound {
		t.Fatalf("update status = %d, want 404", updateRec.Code)
	}

	del := authedRequest(http.MethodDelete, "/api/admin/users/404", "", admin.ID, "admin")
	delRec := recordTestRequest(del, app.handleDeleteUser)
	if delRec.Code != http.StatusNotFound {
		t.Fatalf("delete status = %d, want 404", delRec.Code)
	}
}

func TestAdminUserIDFromPathStrict(t *testing.T) {
	tests := []struct {
		path    string
		want    int64
		wantErr bool
	}{
		{path: "/api/admin/users/12", want: 12},
		{path: "/api/admin/users/", wantErr: true},
		{path: "/api/admin/users/12/extra", wantErr: true},
		{path: "/api/admin/users/abc", wantErr: true},
		{path: "/api/admin/users/0", wantErr: true},
		{path: "/api/admin/user/12", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got, err := adminUserIDFromPath(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got id %d", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("id = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestHandleUpdateUserRejectsExtraPathSegments(t *testing.T) {
	app := newTestAuthApp(t)
	admin := createTestUser(t, app, "admin", model.RoleAdmin)
	target := createTestUser(t, app, "bob", model.RoleUser)

	req := authedRequest(http.MethodPut, "/api/admin/users/"+itoa(target.ID)+"/role", `{"role":"admin","status":1}`, admin.ID, "admin")
	rec := recordTestRequest(req, app.handleUpdateUser)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	got := getTestUser(t, app, target.ID)
	if got.Role != model.RoleUser {
		t.Fatalf("role changed to %q, want unchanged user", got.Role)
	}
}

func TestHandleDeleteUserRejectsExtraPathSegments(t *testing.T) {
	app := newTestAuthApp(t)
	admin := createTestUser(t, app, "admin", model.RoleAdmin)
	target := createTestUser(t, app, "bob", model.RoleUser)

	req := authedRequest(http.MethodDelete, "/api/admin/users/"+itoa(target.ID)+"/force", "", admin.ID, "admin")
	rec := recordTestRequest(req, app.handleDeleteUser)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	getTestUser(t, app, target.ID)
}

// itoa is a tiny local helper to avoid pulling strconv into the test header.
func itoa(n int64) string {
	return strconv.FormatInt(n, 10)
}
