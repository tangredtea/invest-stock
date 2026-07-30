package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"invest/pkg/backtest"
)

func postBacktest(body string) *httptest.ResponseRecorder {
	return postTestJSON("/api/backtest", body, handleBacktest)
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
	decodeTestResponse(t, rec, &resp)
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

func TestHandleBacktestRejectsInvalidFillRule(t *testing.T) {
	seedBacktestData(t, "600519", 200)
	rec := postBacktest(`{"code":"600519","strategy":"买入持有(基准)","config":{"fillRule":99}}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandleBacktestAcceptsTickSlippageConfig(t *testing.T) {
	seedBacktestData(t, "600519", 200)
	rec := postBacktest(`{"code":"600519","strategy":"买入持有(基准)","config":{"slippageMode":1,"tick":0.01,"slippageTicks":2}}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleBacktestRejectsUnknownRequestFields(t *testing.T) {
	seedBacktestData(t, "600519", 200)
	rec := postBacktest(`{"code":"600519","strategy":"买入持有(基准)","unexpected":true}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
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

	rec := postTestJSON("/api/backtest/compare", sb.String(), handleBacktestCompare)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Data []backtest.Result `json:"data"`
	}
	decodeTestResponse(t, rec, &resp)
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
	rec := postTestJSON("/api/backtest/compare", sb.String(), handleBacktestCompare)
	if rec.Code != 400 {
		t.Errorf("expected 400 for >50 combinations, got %d", rec.Code)
	}
}
