package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"invest/internal/auth"
	"invest/pkg/backtest"
)

func postPortfolio(body string) *httptest.ResponseRecorder {
	return postTestJSON("/api/portfolio/backtest", body, handlePortfolioBacktest)
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
	decodeTestResponse(t, rec, &resp)
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

func TestPortfolioBacktestRejectsDuplicateSymbols(t *testing.T) {
	seedPortfolioData(t, "600000", 200, 0)
	rec := postPortfolio(`{"codes":["600000","600000"],"scheme":0}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestPortfolioBacktestRejectsDuplicateSymbolsAfterTrim(t *testing.T) {
	seedPortfolioData(t, "600000", 200, 0)
	rec := postPortfolio(`{"codes":["600000"," 600000 "],"scheme":0}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestPortfolioBacktestNormalizesCodesAndWeightKeys(t *testing.T) {
	seedPortfolioData(t, "600000", 200, 0)
	seedPortfolioData(t, "600001", 200, 1.5)
	body := `{"codes":[" 600000 ","600001"],"scheme":1,"weights":{" 600000 ":0.4,"600001":0.4},"risk":{"perSymbolCap":{" 600000 ":0.5}}}`
	rec := postPortfolio(body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Data backtest.PortfolioResult `json:"data"`
	}
	decodeTestResponse(t, rec, &resp)
	if len(resp.Data.Symbols) != 2 || resp.Data.Symbols[0] != "600000" || resp.Data.Symbols[1] != "600001" {
		t.Fatalf("symbols not normalized/sorted: %+v", resp.Data.Symbols)
	}
	if _, ok := resp.Data.Weights[" 600000 "]; ok {
		t.Fatalf("result contains unnormalized weight key")
	}
	if _, ok := resp.Data.Weights["600000"]; !ok {
		t.Fatalf("result missing normalized weight key")
	}
}

func TestPortfolioBacktestRejectsDuplicateNormalizedWeightKeys(t *testing.T) {
	seedPortfolioData(t, "600000", 200, 0)
	seedPortfolioData(t, "600001", 200, 1.5)
	body := `{"codes":["600000","600001"],"scheme":1,"weights":{"600000":0.3," 600000 ":0.4,"600001":0.3}}`
	rec := postPortfolio(body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestPortfolioBacktestRejectsDuplicateNormalizedCapKeys(t *testing.T) {
	seedPortfolioData(t, "600000", 200, 0)
	seedPortfolioData(t, "600001", 200, 1.5)
	body := `{"codes":["600000","600001"],"scheme":0,"risk":{"perSymbolCap":{"600000":0.4," 600000 ":0.5}}}`
	rec := postPortfolio(body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestPortfolioBacktestRejectsWeightKeyOutsideCodes(t *testing.T) {
	seedPortfolioData(t, "600000", 200, 0)
	seedPortfolioData(t, "600001", 200, 1.5)
	body := `{"codes":["600000","600001"],"scheme":1,"weights":{"600000":0.4,"600002":0.4}}`
	rec := postPortfolio(body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestPortfolioBacktestRejectsCapKeyOutsideCodes(t *testing.T) {
	seedPortfolioData(t, "600000", 200, 0)
	seedPortfolioData(t, "600001", 200, 1.5)
	body := `{"codes":["600000","600001"],"scheme":0,"risk":{"perSymbolCap":{"600002":0.4}}}`
	rec := postPortfolio(body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestPortfolioBacktestInvalidCode(t *testing.T) {
	// Requirement 7.9: invalid security code -> 400.
	rec := postPortfolio(`{"codes":["600000","zzz"],"scheme":0}`)
	if rec.Code != 400 {
		t.Errorf("expected 400 for invalid code, got %d", rec.Code)
	}
}

func TestPortfolioBacktestRejectsUnknownRequestFields(t *testing.T) {
	rec := postPortfolio(`{"codes":["600000","600001"],"scheme":0,"unexpected":true}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
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

func TestPortfolioBacktestInvalidScheme(t *testing.T) {
	seedPortfolioData(t, "600000", 200, 0)
	seedPortfolioData(t, "600001", 200, 1.0)
	rec := postPortfolio(`{"codes":["600000","600001"],"scheme":99}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestPortfolioBacktestRequiresAuth(t *testing.T) {
	// Requirement 7.8: no token -> 401, no backtest run.
	jwtMgr := auth.NewJWTManager(strings.Repeat("k", 40), time.Hour)
	protected := requireAuth(jwtMgr, handlePortfolioBacktest)
	rec := postTestJSON("/api/portfolio/backtest", `{"codes":["600000","600001"],"scheme":0}`, protected)
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
