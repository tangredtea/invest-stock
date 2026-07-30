package indicator

import (
	"math"
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"
)

const eps = 1e-9

// priceSeries is a generated input domain for property tests:
// length 0..1000, each element a finite real in [0.01, 1000000.00].
// (Capped at 1000 rather than 10000 to keep test runtime reasonable while
// still exercising the invariants across many lengths.)
type priceSeries []float64

func (priceSeries) Generate(r *rand.Rand, _ int) reflect.Value {
	n := r.Intn(1001)
	s := make([]float64, n)
	for i := range s {
		s[i] = 0.01 + r.Float64()*(1000000.00-0.01)
	}
	return reflect.ValueOf(priceSeries(s))
}

func isFinite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

// Feature: security-and-performance-hardening, Property 16: 对任意收盘价序列,SMA/RSI/MACD 各分量/
// 布林带各带输出切片长度严格等于输入长度(含输入长度为 0 时输出长度亦为 0)。
func TestProperty16_OutputLengthMatchesInput(t *testing.T) {
	f := func(ps priceSeries) bool {
		n := len(ps)
		closes := []float64(ps)
		if len(SMA(closes, 20)) != n || len(RSI(closes, 14)) != n {
			return false
		}
		ml, sl, hist := MACD(closes, 12, 26, 9)
		if len(ml) != n || len(sl) != n || len(hist) != n {
			return false
		}
		up, mid, low := BollingerBands(closes, 20, 2.0)
		return len(up) == n && len(mid) == n && len(low) == n
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("output length must equal input length: %v", err)
	}
}

// Feature: security-and-performance-hardening, Property 21: 对任意收盘价序列,所有指标输出切片中的
// 每个位置都是有限实数(不含 NaN 与 ±Inf)。
func TestProperty21_OutputsAreFinite(t *testing.T) {
	f := func(ps priceSeries) bool {
		closes := []float64(ps)
		series := [][]float64{SMA(closes, 20), RSI(closes, 14)}
		ml, sl, hist := MACD(closes, 12, 26, 9)
		up, mid, low := BollingerBands(closes, 20, 2.0)
		series = append(series, ml, sl, hist, up, mid, low)
		for _, s := range series {
			for _, v := range s {
				if !isFinite(v) {
					return false
				}
			}
		}
		return true
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("all indicator outputs must be finite: %v", err)
	}
}

// Feature: security-and-performance-hardening, Property 22: 对任意收盘价序列与回看周期,首个有效窗口
// 位置之前的所有前导位置(及序列短于回看周期的情形)产生精确为 0 的零填充值且不引发运行时错误。
func TestProperty22_LeadingZeroFill(t *testing.T) {
	f := func(ps priceSeries) bool {
		closes := []float64(ps)
		// SMA: first period-1 entries are exactly 0.
		const smaP = 20
		sma := SMA(closes, smaP)
		for i := 0; i < len(sma) && i < smaP-1; i++ {
			if sma[i] != 0 {
				return false
			}
		}
		// RSI: first period entries are exactly 0.
		const rsiP = 14
		rsi := RSI(closes, rsiP)
		for i := 0; i < len(rsi) && i < rsiP; i++ {
			if rsi[i] != 0 {
				return false
			}
		}
		// Bollinger: first period-1 entries of upper/lower are exactly 0.
		const bollP = 20
		up, _, low := BollingerBands(closes, bollP, 2.0)
		for i := 0; i < len(up) && i < bollP-1; i++ {
			if up[i] != 0 || low[i] != 0 {
				return false
			}
		}
		return true
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("leading positions must be zero-filled: %v", err)
	}
}

func TestProperty23_AtEqualsComputeAll(t *testing.T) {
	f := func(ps priceSeries) bool {
		closes := []float64(ps)
		series := ComputeSeries(closes)
		for i := range closes {
			if At(series, i) != ComputeAll(closes, i) {
				return false
			}
		}
		return true
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("At must equal ComputeAll field-for-field: %v", err)
	}
}

func TestComputeAllHandlesOutOfRangeIndex(t *testing.T) {
	tests := []struct {
		name   string
		closes []float64
		index  int
	}{
		{name: "empty", closes: nil, index: 0},
		{name: "negative index", closes: []float64{1, 2, 3}, index: -1},
		{name: "too large", closes: []float64{1, 2, 3}, index: 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ComputeAll(tt.closes, tt.index); got != (Indicators{}) {
				t.Fatalf("ComputeAll = %+v, want zero Indicators", got)
			}
		})
	}
}

func TestAtHandlesOutOfRangeIndex(t *testing.T) {
	series := ComputeSeries([]float64{1, 2, 3})
	for _, index := range []int{-1, 3} {
		if got := At(series, index); got != (Indicators{}) {
			t.Fatalf("At index %d = %+v, want zero Indicators", index, got)
		}
	}
}

func TestIndicatorsHandleNonPositivePeriods(t *testing.T) {
	closes := []float64{1, 2, 3, 4, 5}
	if got := SMA(closes, 0); len(got) != len(closes) {
		t.Fatalf("SMA length = %d, want %d", len(got), len(closes))
	}
	if got := RSI(closes, -1); len(got) != len(closes) {
		t.Fatalf("RSI length = %d, want %d", len(got), len(closes))
	}
	if got := EMA(closes, 0); len(got) != len(closes) {
		t.Fatalf("EMA length = %d, want %d", len(got), len(closes))
	}
	ml, sl, hist := MACD(closes, 0, 26, 9)
	if len(ml) != len(closes) || len(sl) != len(closes) || len(hist) != len(closes) {
		t.Fatalf("MACD lengths = %d/%d/%d, want %d", len(ml), len(sl), len(hist), len(closes))
	}
	up, mid, low := BollingerBands(closes, 0, 2)
	if len(up) != len(closes) || len(mid) != len(closes) || len(low) != len(closes) {
		t.Fatalf("Bollinger lengths = %d/%d/%d, want %d", len(up), len(mid), len(low), len(closes))
	}
	for _, series := range [][]float64{SMA(closes, 0), RSI(closes, -1), EMA(closes, 0), ml, sl, hist, up, mid, low} {
		for _, v := range series {
			if v != 0 {
				t.Fatalf("invalid-period output contains non-zero value: %v", series)
			}
		}
	}
}
