package main

import (
	"errors"
	"testing"

	"invest/pkg/data"
)

func TestResolveSecIDForCode(t *testing.T) {
	secid, clientErr := resolveSecIDForCode("600519")
	if clientErr != klineClientErrorNone {
		t.Fatalf("clientErr = %v, want none", clientErr)
	}
	if secid != "1.600519" {
		t.Fatalf("secid = %q, want 1.600519", secid)
	}

	secid, clientErr = resolveSecIDForCode("bad")
	if clientErr != klineClientErrorInvalidCode {
		t.Fatalf("clientErr = %v, want invalid code", clientErr)
	}
	if secid != "" {
		t.Fatalf("secid = %q, want empty", secid)
	}
}

func TestLoadKLinesForCodeRejectsInvalidCode(t *testing.T) {
	_, clientErr, err := loadKLinesForCode("bad")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clientErr != klineClientErrorInvalidCode {
		t.Fatalf("clientErr = %v, want invalid code", clientErr)
	}
}

func TestLoadKLinesForCodeClassifiesFetchError(t *testing.T) {
	stubFetchKLines(t, func(string) ([]data.KLine, error) {
		return nil, errors.New("upstream unavailable")
	})

	_, clientErr, err := loadKLinesForCode("600519")
	if err == nil {
		t.Fatal("expected fetch error")
	}
	if clientErr != klineClientErrorFetchFailed {
		t.Fatalf("clientErr = %v, want fetch failed", clientErr)
	}
}

func TestLoadKLinesForCodeReturnsKLines(t *testing.T) {
	seedBacktestData(t, "600519", 120)

	loaded, clientErr, err := loadKLinesForCode("600519")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clientErr != klineClientErrorNone {
		t.Fatalf("clientErr = %v, want none", clientErr)
	}
	if loaded.SecID != "1.600519" {
		t.Fatalf("secid = %q, want 1.600519", loaded.SecID)
	}
	if len(loaded.KLines) != 120 {
		t.Fatalf("klines = %d, want 120", len(loaded.KLines))
	}
}
