package main

import (
	"math"
	"testing"
	"time"

	"invest/pkg/data"
)

func seedTestKLines(t *testing.T, code string, n int, price func(int) float64, spread float64) {
	t.Helper()
	secid, clientErr := resolveSecIDForCode(code)
	if clientErr != klineClientErrorNone {
		t.Fatalf("bad code %q", code)
	}
	base := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)
	klines := make([]data.KLine, n)
	for i := 0; i < n; i++ {
		p := price(i)
		klines[i] = data.KLine{
			Date:   base.AddDate(0, 0, i),
			Open:   p,
			High:   p + spread,
			Low:    p - spread,
			Close:  p,
			Volume: 1000,
		}
	}
	data.SeedKLineCache(secid, klines)
}

func seedBacktestData(t *testing.T, code string, n int) {
	t.Helper()
	seedTestKLines(t, code, n, func(i int) float64 {
		return 10 + 2*math.Sin(float64(i)/7)
	}, 0.3)
}

func seedPortfolioData(t *testing.T, code string, n int, phase float64) {
	t.Helper()
	seedTestKLines(t, code, n, func(i int) float64 {
		return 10 + 3*math.Sin(float64(i)/9+phase)
	}, 0.2)
}
