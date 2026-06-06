package main

import (
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"invest/pkg/backtest"
	"invest/pkg/data"
)

// seedBacktestData preloads the cache with a deterministic kline series for the
// given code's secid so the backtest handlers don't hit the network.
func seedBacktestData(t *testing.T, code string, n int) {
	t.Helper()
	secid := resolveSecID(code)
	if secid == "" {
		t.Fatalf("bad code %q", code)
	}
	base := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)
	klines := make([]data.KLine, n)
	for i := 0; i < n; i++ {
		price := 10 + 2*math.Sin(float64(i)/7)
		klines[i] = data.KLine{
			Date: base.AddDate(0, 0, i), Open: price, High: price + 0.3,
			Low: price - 0.3, Close: price, Volume: 1000,
		}
	}
	data.SeedKLineCache(secid, klines)
}

func postBacktest(body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/backtest", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handleBacktest(rec, req)
	return rec
}

func TestHandleBacktestSuccess(t *testing.T) {
	// Requirement 14.2: a valid request returns a full Result.
	seedBacktestData(t, "600519", 200)
	body := `{"code":"600519","strategy":"` + backtest.Default.List()[0].Name + `"}`
	// Use a known builtin name via the registry list to avoid hardcoding.
	rec := postBacktest(body)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Code int             `json:"code"`
		Data backtest.Result `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Data.Equity) == 0 || len(resp.Data.Drawdown) == 0 {
		t.Errorf("expected equity/drawdown curves in result")
	}
}

func TestHandleBacktestUnknownStrategy(t *testing.T) {
	// Requirement 14.5: unregistered strategy -> 400.
	seedBacktestData(t, "600519", 200)
	rec := postBacktest(`{"code":"600519","strategy":"不存在的策略"}`)
	if rec.Code != 400 {
		t.Errorf("expected 400 for unknown strategy, got %d", rec.Code)
	}
}

func TestHandleBacktestInvalidCode(t *testing.T) {
	// Requirement 14.7: invalid security code -> 400.
	rec := postBacktest(`{"code":"abc","strategy":"买入持有(基准)"}`)
	if rec.Code != 400 {
		t.Errorf("expected 400 for invalid code, got %d", rec.Code)
	}
}

func TestHandleBacktestInvalidConfig(t *testing.T) {
	// Requirement 14.6: non-positive initial cash -> 400.
	seedBacktestData(t, "600519", 200)
	rec := postBacktest(`{"code":"600519","strategy":"买入持有(基准)","config":{"initialCash":-1}}`)
	if rec.Code != 400 {
		t.Errorf("expected 400 for invalid config, got %d", rec.Code)
	}
}

func TestHandleBacktestCompareNineBuiltins(t *testing.T) {
	// Requirement 15.5: comparing all 9 builtins returns 9 results.
	seedBacktestData(t, "600519", 200)
	var sb strings.Builder
	sb.WriteString(`{"code":"600519","strategies":[`)
	for i, info := range backtest.Default.List() {
		if i > 0 {
			sb.WriteString(",")
		}
		b, _ := json.Marshal(info.Name)
		sb.WriteString(`{"strategy":` + string(b) + `}`)
	}
	sb.WriteString(`]}`)

	req := httptest.NewRequest(http.MethodPost, "/api/backtest/compare", strings.NewReader(sb.String()))
	rec := httptest.NewRecorder()
	handleBacktestCompare(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Data []backtest.Result `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Data) != 9 {
		t.Errorf("expected 9 results, got %d", len(resp.Data))
	}
}

func TestHandleBacktestCompareTooMany(t *testing.T) {
	// Requirement 15.3: more than 50 combinations -> 400.
	seedBacktestData(t, "600519", 200)
	var sb strings.Builder
	sb.WriteString(`{"code":"600519","strategies":[`)
	for i := 0; i < 51; i++ {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(`{"strategy":"买入持有(基准)"}`)
	}
	sb.WriteString(`]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/backtest/compare", strings.NewReader(sb.String()))
	rec := httptest.NewRecorder()
	handleBacktestCompare(rec, req)
	if rec.Code != 400 {
		t.Errorf("expected 400 for >50 combinations, got %d", rec.Code)
	}
}
