package main

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"invest/internal/auth"
)

// TestBacktestRequiresAuth verifies the backtest endpoint is protected by
// requireAuth: a request without a token returns 401 and does not run a
// backtest (Requirement 14.4).
func TestBacktestRequiresAuth(t *testing.T) {
	jwtMgr := auth.NewJWTManager(strings.Repeat("k", 40), time.Hour)
	protected := requireAuth(jwtMgr, handleBacktest)

	rec := postTestJSON("/api/backtest", `{"code":"600519","strategy":"买入持有(基准)"}`, protected)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without token, got %d", rec.Code)
	}
}

func TestBacktestWithValidTokenPasses(t *testing.T) {
	// With a valid token, requireAuth delegates to the handler (which then
	// processes the request). We only assert it is NOT a 401.
	jwtMgr := auth.NewJWTManager(strings.Repeat("k", 40), time.Hour)
	token, err := jwtMgr.Generate(1, "alice", "user")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	seedBacktestData(t, "600519", 200)
	protected := requireAuth(jwtMgr, handleBacktest)

	rec := postTestJSON("/api/backtest", `{"code":"600519","strategy":"买入持有(基准)"}`, func(w http.ResponseWriter, r *http.Request) {
		r.Header.Set("Authorization", "Bearer "+token)
		protected(w, r)
	})

	if rec.Code == http.StatusUnauthorized {
		t.Errorf("valid token should not yield 401")
	}
}
