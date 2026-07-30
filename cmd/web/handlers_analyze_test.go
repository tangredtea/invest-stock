package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"invest/pkg/data"
)

func TestAnalyzeAndQuoteErrorsAreJSON(t *testing.T) {
	cases := []struct {
		name    string
		handler func(http.ResponseWriter, *http.Request)
		target  string
	}{
		{"analyze invalid code", handleAnalyze, "/api/analyze?code=bad"},
		{"quote invalid code", handleQuote, "/api/quote?code=bad"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := callTestHandler(http.MethodGet, tc.target, "", tc.handler)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", rec.Code)
			}
			if got := rec.Header().Get("Content-Type"); got != "application/json" {
				t.Fatalf("Content-Type = %q, want application/json", got)
			}
			var body map[string]string
			decodeTestResponse(t, rec, &body)
			if body["error"] == "" {
				t.Fatal("expected non-empty error field")
			}
		})
	}
}

func TestAnalyzeSurfacesQuoteFetchFailures(t *testing.T) {
	mux := newTestApp(t)
	token := registerTestUser(t, mux, "quotewarn")
	seedBacktestData(t, "600519", 120)
	stubFetchQuote(t, func(string) (data.Quote, error) {
		return data.Quote{}, errors.New("quote unavailable")
	})

	req := authRequest(http.MethodGet, "/api/analyze?code=600519", token, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		QuoteWarning string `json:"quoteWarning"`
	}
	decodeTestResponse(t, rec, &resp)
	if resp.QuoteWarning == "" {
		t.Fatal("expected quoteWarning to be populated when quote fetch fails")
	}
}

func TestAnalyzeRejectsInsufficientKLines(t *testing.T) {
	stubFetchKLines(t, func(string) ([]data.KLine, error) {
		return []data.KLine{{Close: 1}}, nil
	})

	rec := callTestHandler(http.MethodGet, "/api/analyze?code=600519", "", handleAnalyze)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	var body map[string]string
	decodeTestResponse(t, rec, &body)
	if body["error"] == "" {
		t.Fatal("expected error message")
	}
}

func TestLoadAnalyzeDataClassifiesInvalidCode(t *testing.T) {
	_, clientErr, err := loadAnalyzeData("bad")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clientErr != analyzeClientErrorInvalidCode {
		t.Fatalf("clientErr = %v, want invalid code", clientErr)
	}
}

func TestLoadAnalyzeDataClassifiesInsufficientKLines(t *testing.T) {
	stubFetchKLines(t, func(string) ([]data.KLine, error) {
		return []data.KLine{{Close: 1}}, nil
	})

	_, clientErr, err := loadAnalyzeData("600519")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clientErr != analyzeClientErrorInsufficientKLines {
		t.Fatalf("clientErr = %v, want insufficient K-lines", clientErr)
	}
}

func TestLoadAnalyzeDataReturnsSnapshot(t *testing.T) {
	seedBacktestData(t, "600519", 120)

	loaded, clientErr, err := loadAnalyzeData("600519")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clientErr != analyzeClientErrorNone {
		t.Fatalf("clientErr = %v, want none", clientErr)
	}
	if loaded.SecID != "1.600519" || len(loaded.Klines) != 120 || len(loaded.Snapshot.Closes) != 120 {
		t.Fatalf("unexpected loaded data: secid=%q klines=%d closes=%d", loaded.SecID, len(loaded.Klines), len(loaded.Snapshot.Closes))
	}
}
