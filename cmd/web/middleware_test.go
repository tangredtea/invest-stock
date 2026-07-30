package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"invest/internal/auth"
)

func TestRequireAuthRejectsQueryToken(t *testing.T) {
	jwtMgr := auth.NewJWTManager(strings.Repeat("k", 40), time.Hour)
	token, err := jwtMgr.Generate(1, "alice", "user")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	called := false
	protected := requireAuth(jwtMgr, func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	req := httptest.NewRequest(http.MethodGet, "/api/auth/me?token="+token, nil)
	rec := httptest.NewRecorder()
	protected(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if called {
		t.Fatal("handler should not be called when token is only in query string")
	}
}

func TestRequireAuthAcceptsBearerHeader(t *testing.T) {
	jwtMgr := auth.NewJWTManager(strings.Repeat("k", 40), time.Hour)
	token, err := jwtMgr.Generate(1, "alice", "user")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	called := false
	protected := requireAuth(jwtMgr, func(w http.ResponseWriter, r *http.Request) {
		called = true
		writeJSON(w, http.StatusOK, jsonResp{Code: 0, Message: "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	protected(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !called {
		t.Fatal("handler should be called for valid bearer token")
	}
}
