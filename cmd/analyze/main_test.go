package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"invest/pkg/data"
)

func TestAnalyzeKLinesRequiresEnoughData(t *testing.T) {
	if _, err := analyzeKLines(makeKLines(60)); err == nil {
		t.Fatal("expected error for insufficient data")
	}
}

func TestAnalyzeKLinesRejectsZeroClose(t *testing.T) {
	klines := makeKLines(61)
	klines[60].Close = 0
	if _, err := analyzeKLines(klines); err == nil {
		t.Fatal("expected error for zero close")
	}
}

func TestAnalyzeKLinesComputesSpreadAndProfitStats(t *testing.T) {
	klines := makeKLines(64)
	for i := 60; i < 64; i++ {
		klines[i] = data.KLine{
			Date:   time.Date(2024, 1, i-58, 0, 0, 0, 0, time.UTC),
			Open:   100,
			High:   float64(101 + (i - 60)),
			Low:    99,
			Close:  100,
			Volume: 1000,
		}
	}

	got, err := analyzeKLines(klines)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Count != 4 {
		t.Fatalf("Count = %d, want 4", got.Count)
	}
	if got.Average != 3.5 || got.Median != 4 || got.P25 != 3 || got.P75 != 5 || got.Max != 5 {
		t.Fatalf("unexpected spread stats: %+v", got)
	}
	if got.TotalProfit != 4000 || got.DailyProfit != 1000 || got.WinRate != 100 {
		t.Fatalf("unexpected profit stats: %+v", got)
	}
	if len(got.Thresholds) != 4 || got.Thresholds[0].Count != 4 {
		t.Fatalf("unexpected threshold stats: %+v", got.Thresholds)
	}
}

func TestWriteAnalysisUsesProvidedWriter(t *testing.T) {
	var buf bytes.Buffer
	writeAnalysis(&buf, analysisResult{
		Count:       2,
		Average:     1.2,
		Median:      1.3,
		P25:         1.1,
		P75:         1.4,
		Max:         1.4,
		Thresholds:  []thresholdStat{{Threshold: 1, Count: 1, Percent: 50}},
		TotalProfit: 100,
		DailyProfit: 50,
		WinRate:     50,
	})

	out := buf.String()
	for _, want := range []string{"日内振幅分析", "振幅>=1.0%", "T+0理想模拟", "胜率: 50%"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q: %s", want, out)
		}
	}
	if strings.Contains(out, "NaN") || strings.Contains(out, "Inf") {
		t.Fatalf("output contains non-finite value: %s", out)
	}
}

func makeKLines(n int) []data.KLine {
	klines := make([]data.KLine, n)
	for i := range klines {
		klines[i] = data.KLine{
			Date:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, i),
			Open:   1,
			High:   1.2,
			Low:    0.9,
			Close:  1.1,
			Volume: 1000,
		}
	}
	return klines
}
