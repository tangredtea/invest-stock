package main

import (
	"bytes"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"invest/internal/auth"
	"invest/internal/model"
	"invest/internal/store"
	"invest/pkg/backtest"
	"invest/pkg/data"
)

// TestHTTPServerTimeoutsValuesMatchSpec is a smoke test pinning the timeout
// values against Requirement 8.1 (read 15s, write 15s, idle 60s, read-header 5s).
func TestHTTPServerTimeoutsValuesMatchSpec(t *testing.T) {
	srv := newHTTPServer(":0", http.NewServeMux())
	if srv.ReadTimeout != 15*time.Second {
		t.Errorf("ReadTimeout = %v, want 15s", srv.ReadTimeout)
	}
	if srv.WriteTimeout != 15*time.Second {
		t.Errorf("WriteTimeout = %v, want 15s", srv.WriteTimeout)
	}
	if srv.IdleTimeout != 60*time.Second {
		t.Errorf("IdleTimeout = %v, want 60s", srv.IdleTimeout)
	}
	if srv.ReadHeaderTimeout != 5*time.Second {
		t.Errorf("ReadHeaderTimeout = %v, want 5s", srv.ReadHeaderTimeout)
	}
}

func TestNewAppMuxServesStaticFiles(t *testing.T) {
	mux := newTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/static/css/common.css", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("static status = %d, want 200", rec.Code)
	}
}

func newTestApp(t *testing.T) http.Handler {
	mux, _, _ := newTestAppWithAuth(t)
	return mux
}

func newTestAppWithAuth(t *testing.T) (http.Handler, *model.UserStore, *auth.JWTManager) {
	t.Helper()
	db, err := store.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	tmplSub, err := fs.Sub(templateFiles, "templates")
	if err != nil {
		t.Fatalf("load templates: %v", err)
	}
	renderer, err := NewRenderer(tmplSub)
	if err != nil {
		t.Fatalf("create renderer: %v", err)
	}

	jwtMgr := auth.NewJWTManager(strings.Repeat("k", 40), time.Hour)
	users := model.NewUserStore(db.DB)
	mux, err := newAppMux(appDeps{
		jwt:      jwtMgr,
		users:    users,
		limiter:  NewLoginRateLimiter(defaultLoginThreshold, defaultLoginWindow),
		renderer: renderer,
	})
	if err != nil {
		t.Fatalf("create app mux: %v", err)
	}
	return mux, users, jwtMgr
}

func registerTestUser(t *testing.T, mux http.Handler, username string) string {
	t.Helper()
	registerBody := bytes.NewBufferString(`{"username":"` + username + `","password":"strongpass"}`)
	registerReq := httptest.NewRequest(http.MethodPost, "/api/auth/register", registerBody)
	registerReq.Header.Set("Content-Type", "application/json")
	registerRec := httptest.NewRecorder()
	mux.ServeHTTP(registerRec, registerReq)
	if registerRec.Code != http.StatusOK {
		t.Fatalf("register status = %d, body = %s", registerRec.Code, registerRec.Body.String())
	}

	var registerResp struct {
		Code int `json:"code"`
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	decodeTestResponse(t, registerRec, &registerResp)
	if registerResp.Code != 0 || registerResp.Data.Token == "" {
		t.Fatalf("invalid register response: %+v", registerResp)
	}
	return registerResp.Data.Token
}

func authRequest(method, target, token string, body *bytes.Buffer) *http.Request {
	if body == nil {
		body = bytes.NewBuffer(nil)
	}
	req := httptest.NewRequest(method, target, body)
	req.Header.Set("Authorization", "Bearer "+token)
	if body.Len() > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

func TestAppE2ERegisterAndReadCurrentUser(t *testing.T) {
	mux := newTestApp(t)
	token := registerTestUser(t, mux, "alice")

	req := authRequest(http.MethodGet, "/api/auth/me", token, nil)
	meRec := httptest.NewRecorder()
	mux.ServeHTTP(meRec, req)
	if meRec.Code != http.StatusOK {
		t.Fatalf("me status = %d, body = %s", meRec.Code, meRec.Body.String())
	}
}

func TestAppE2EBacktestAPIThroughMux(t *testing.T) {
	mux := newTestApp(t)
	token := registerTestUser(t, mux, "bob")
	seedBacktestData(t, "600519", 200)

	body := bytes.NewBufferString(`{"code":"600519","strategy":"买入持有(基准)","config":{"initialCash":100000,"slippageMode":1,"tick":0.01,"slippageTicks":1}}`)
	req := authRequest(http.MethodPost, "/api/backtest", token, body)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("backtest status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Code int             `json:"code"`
		Data backtest.Result `json:"data"`
	}
	decodeTestResponse(t, rec, &resp)
	if resp.Code != 0 || len(resp.Data.Equity) == 0 || len(resp.Data.Dates) == 0 {
		t.Fatalf("unexpected backtest response: code=%d equity=%d dates=%d", resp.Code, len(resp.Data.Equity), len(resp.Data.Dates))
	}
}

func TestAppE2EPortfolioAPIThroughMux(t *testing.T) {
	mux := newTestApp(t)
	token := registerTestUser(t, mux, "carol")
	seedPortfolioData(t, "600000", 200, 0)
	seedPortfolioData(t, "600001", 200, 1.5)

	body := bytes.NewBufferString(`{"codes":["600000","600001"],"scheme":0,"rebalance":{"periodic":true,"periodBars":20}}`)
	req := authRequest(http.MethodPost, "/api/portfolio/backtest", token, body)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("portfolio status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Code int                      `json:"code"`
		Data backtest.PortfolioResult `json:"data"`
	}
	decodeTestResponse(t, rec, &resp)
	if resp.Code != 0 || len(resp.Data.Equity) == 0 || len(resp.Data.Weights) != 2 {
		t.Fatalf("unexpected portfolio response: code=%d equity=%d weights=%d", resp.Code, len(resp.Data.Equity), len(resp.Data.Weights))
	}
}

func TestAppE2EStrategiesRequiresAuth(t *testing.T) {
	mux := newTestApp(t)

	unauth := httptest.NewRequest(http.MethodGet, "/api/strategies", nil)
	unauthRec := httptest.NewRecorder()
	mux.ServeHTTP(unauthRec, unauth)
	if unauthRec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d, want 401", unauthRec.Code)
	}

	token := registerTestUser(t, mux, "dave")
	req := authRequest(http.MethodGet, "/api/strategies", token, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("strategies status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Code int                     `json:"code"`
		Data []backtest.StrategyInfo `json:"data"`
	}
	decodeTestResponse(t, rec, &resp)
	if resp.Code != 0 || len(resp.Data) == 0 {
		t.Fatalf("unexpected strategies response: %+v", resp)
	}
}

func TestAppE2EAPIsRejectUnexpectedMethods(t *testing.T) {
	mux, users, jwtMgr := newTestAppWithAuth(t)
	admin, err := users.Create("methodadmin", "strongpass", model.RoleAdmin)
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}
	token, err := jwtMgr.Generate(admin.ID, admin.Username, admin.Role)
	if err != nil {
		t.Fatalf("generate admin token: %v", err)
	}

	tests := []struct {
		method string
		path   string
		allow  string
	}{
		{method: http.MethodPost, path: "/api/auth/me", allow: http.MethodGet},
		{method: http.MethodPost, path: "/api/strategies", allow: http.MethodGet},
		{method: http.MethodPost, path: "/api/quote?code=600519", allow: http.MethodGet},
		{method: http.MethodPost, path: "/api/analyze?code=600519", allow: http.MethodGet},
		{method: http.MethodPost, path: "/api/admin/users/1", allow: "PUT, DELETE"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := authRequest(tt.method, tt.path, token, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			if rec.Code != http.StatusMethodNotAllowed {
				t.Fatalf("status = %d, want 405: %s", rec.Code, rec.Body.String())
			}
			if got := rec.Header().Get("Allow"); got != tt.allow {
				t.Fatalf("Allow = %q, want %q", got, tt.allow)
			}
		})
	}
}

func TestAppE2EAdminUserDetailRejectsExtraPathSegments(t *testing.T) {
	mux, users, jwtMgr := newTestAppWithAuth(t)
	admin, err := users.Create("admin", "strongpass", model.RoleAdmin)
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}
	target, err := users.Create("target", "strongpass", model.RoleUser)
	if err != nil {
		t.Fatalf("create target: %v", err)
	}
	token, err := jwtMgr.Generate(admin.ID, admin.Username, admin.Role)
	if err != nil {
		t.Fatalf("generate admin token: %v", err)
	}

	body := bytes.NewBufferString(`{"role":"admin","status":1}`)
	req := authRequest(http.MethodPut, "/api/admin/users/"+strconv.FormatInt(target.ID, 10)+"/role", token, body)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", rec.Code, rec.Body.String())
	}

	got, err := users.GetByID(target.ID)
	if err != nil {
		t.Fatalf("get target: %v", err)
	}
	if got.Role != model.RoleUser {
		t.Fatalf("role changed to %q, want unchanged user", got.Role)
	}
}

func TestAppE2EQuoteAPIThroughMux(t *testing.T) {
	mux := newTestApp(t)
	token := registerTestUser(t, mux, "frank")
	stubFetchQuote(t, func(secid string) (data.Quote, error) {
		if secid != "1.600519" {
			t.Fatalf("secid = %q, want 1.600519", secid)
		}
		return data.Quote{Price: 10.5, Open: 10.2, High: 10.8, Low: 10.1, PreClose: 10}, nil
	})

	req := authRequest(http.MethodGet, "/api/quote?code=600519", token, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("quote status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var resp data.Quote
	decodeTestResponse(t, rec, &resp)
	if resp.Price != 10.5 || resp.PreClose != 10 {
		t.Fatalf("unexpected quote response: %+v", resp)
	}
}

func TestAppE2EAnalyzeUsesSeededKLinesAndGracefulQuoteFallback(t *testing.T) {
	mux := newTestApp(t)
	token := registerTestUser(t, mux, "erin")
	secid, clientErr := resolveSecIDForCode("600519")
	if clientErr != klineClientErrorNone {
		t.Fatalf("resolve test secid failed: %v", clientErr)
	}
	seedBacktestData(t, "600519", 120)
	stubFetchQuote(t, func(string) (data.Quote, error) {
		return data.Quote{}, errors.New("quote unavailable")
	})

	req := authRequest(http.MethodGet, "/api/analyze?code=600519", token, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("analyze status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var resp analyzeResp
	decodeTestResponse(t, rec, &resp)
	if resp.SecID != secid || len(resp.Klines) != 120 || len(resp.Indicators.MA5) != 120 {
		t.Fatalf("unexpected analyze response: secid=%q klines=%d ma5=%d", resp.SecID, len(resp.Klines), len(resp.Indicators.MA5))
	}
}
