package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"invest/pkg/data"
	"invest/pkg/strategy"
)

func TestPrintHeaderFallsBackForInvalidQuote(t *testing.T) {
	klines := []data.KLine{{
		Date:  time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		Close: 1.23,
	}}
	q := data.Quote{Price: 1.25, Open: 1.20, High: 1.30, Low: 1.10, PreClose: 0}

	var buf bytes.Buffer
	printHeaderTo(&buf, klines, q)

	out := buf.String()
	if !strings.Contains(out, "收盘: 1.230") {
		t.Fatalf("output = %q, want close fallback", out)
	}
	if strings.Contains(out, "Inf") || strings.Contains(out, "NaN") {
		t.Fatalf("output contains non-finite percentage: %q", out)
	}
}

func TestValidQuoteForDisplayRejectsInconsistentQuotes(t *testing.T) {
	valid := data.Quote{Price: 1.25, Open: 1.20, High: 1.30, Low: 1.10, PreClose: 1.22}
	if !validQuoteForDisplay(valid) {
		t.Fatal("valid quote rejected")
	}
	tests := []data.Quote{
		{Price: 1.25, Open: 1.20, High: 1.30, Low: 1.10, PreClose: 0},
		{Price: 1.25, Open: 1.20, High: 1.00, Low: 1.10, PreClose: 1.22},
		{Price: 1.25, Open: 1.20, High: 1.30, Low: 1.40, PreClose: 1.22},
	}
	for _, q := range tests {
		if validQuoteForDisplay(q) {
			t.Fatalf("invalid quote accepted: %+v", q)
		}
	}
}

func TestSignalOutputWritesToProvidedWriter(t *testing.T) {
	sig := strategy.SignalResult{
		Trend:     strategy.TrendUp,
		DCASignal: strategy.Hold,
		T0Dir:     strategy.T0Skip,
		Reason:    "测试信号",
		T0Reasons: []string{"测试依据"},
	}
	var buf bytes.Buffer
	printSignalTo(&buf, sig)

	out := buf.String()
	if !strings.Contains(out, "【定投信号】") || !strings.Contains(out, "【T+0操作建议】") {
		t.Fatalf("missing expected sections: %q", out)
	}
	if strings.Contains(out, "NaN") || strings.Contains(out, "Inf") {
		t.Fatalf("unexpected non-finite output: %q", out)
	}
}
