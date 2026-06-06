package main

import (
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"invest/internal/auth"
	"invest/pkg/backtest"
	"invest/pkg/data"
)

func seedPortfolioData(t *testing.T, code string, n int, phase float64) {
	t.Helper()
	secid := resolveSecID(code)
	if secid == "" {
		t.Fatalf("bad code %q", code)
	}
	base := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)
	klines := make([]data.KLine, n)
	for i := 0; i < n; i++ {
		price := 10 + 3*math.Sin(float64(i)/9+phase)
		klines[i] = data.KLine{
			Date: base.AddDate(0, 0, i), Open: price, High: price + 0.2,
			Low: price - 0.2, Close: price, Volume: 1000,
		}
	}
	data.SeedKLineCache(secid, klines)
}

func postPortfolio(body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/portfolio/backtest", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handlePortfolioBacktest(rec, req)
	return rec
}

func TestPortfolioBacktestSuccess(t *testing.T) {
	// Requirements 7.2, 7.3: valid request returns full portfolio result.
	seedPortfolioData(t, "600000", 200, 0)
	seedPortfolioData(t, "600001", 200, 1.5)
	seedPortfolioData(t, "600002", 200, 3.0)
	body := `{"codes":["600000","600001","600002"],"scheme":0,"rebalance":{"periodic":true,"periodBars":20}}`
	rec := postPortfolio(body)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Code int                      `json:"code"`
		Data backtest.PortfolioResult `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Data.Equity) == 0 || len(resp.Data.Weights) != 3 {
		t.Errorf("expected equity curve and 3 symbol weight series, got equity=%d weights=%d", len(resp.Data.Equity), len(resp.Data.Weights))
	}
	if len(resp.Data.Attribution) != 3 {
		t.Errorf("expected 3 attribution entries, got %d", len(resp.Data.Attribution))
	}
}

func TestPortfolioBacktestTooFewSymbols(t *testing.T) {
	// Requirement 7.4: fewer than 2 symbols -> 400.
	seedPortfolioData(t, "600000", 200, 0)
	rec := postPortfolio(`{"codes":["600000"],"scheme":0}`)
	if rec.Code != 400 {
		t.Errorf("expected 400 for single symbol, got %d", rec.Code)
	}
}

func TestPortfolioBacktestInvalidCode(t *testing.T) {
	// Requirement 7.9: invalid security code -> 400.
	rec := postPortfolio(`{"codes":["600000","zzz"],"scheme":0}`)
	if rec.Code != 400 {
		t.Errorf("expected 400 for invalid code, got %d", rec.Code)
	}
}

func TestPortfolioBacktestInvalidConfig(t *testing.T) {
	// Requirement 5.10 via 7.4: invalid cash reserve -> 400.
	seedPortfolioData(t, "600000", 200, 0)
	seedPortfolioData(t, "600001", 200, 1.0)
	rec := postPortfolio(`{"codes":["600000","600001"],"scheme":0,"risk":{"cashReserve":2.0}}`)
	if rec.Code != 400 {
		t.Errorf("expected 400 for invalid cash reserve, got %d", rec.Code)
	}
}

func TestPortfolioBacktestRequiresAuth(t *testing.T) {
	// Requirement 7.8: no token -> 401, no backtest run.
	jwtMgr := auth.NewJWTManager(strings.Repeat("k", 40), time.Hour)
	protected := requireAuth(jwtMgr, handlePortfolioBacktest)
	req := httptest.NewRequest(http.MethodPost, "/api/portfolio/backtest",
		strings.NewReader(`{"codes":["600000","600001"],"scheme":0}`))
	rec := httptest.NewRecorder()
	protected(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without token, got %d", rec.Code)
	}
}

func TestPortfolioBacktestPerSymbolAdapter(t *testing.T) {
	// PerSymbol mode wraps a builtin strategy across symbols.
	seedPortfolioData(t, "600000", 200, 0)
	seedPortfolioData(t, "600001", 200, 2.0)
	body := `{"codes":["600000","600001"],"perSymbol":true,"strategy":"双均线交叉(MA5/20)"}`
	rec := postPortfolio(body)
	if rec.Code != 200 {
		t.Fatalf("expected 200 for per-symbol adapter, got %d: %s", rec.Code, rec.Body.String())
	}
}
