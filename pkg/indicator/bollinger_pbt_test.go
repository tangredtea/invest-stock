package indicator

import (
	"math"
	"testing"
	"testing/quick"
)

// Feature: security-and-performance-hardening, Property 19: 对任意收盘价序列,布林带在每个有效窗口
// 位置(i ≥ P−1)满足上轨 ≥ 中轨 ≥ 下轨(eps 容差)。
func TestProperty19_BollingerBandsOrdered(t *testing.T) {
	const period = 20
	f := func(ps priceSeries) bool {
		closes := []float64(ps)
		up, mid, low := BollingerBands(closes, period, 2.0)
		for i := period - 1; i < len(closes); i++ {
			if up[i] < mid[i]-eps || mid[i] < low[i]-eps {
				return false
			}
		}
		return true
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("Bollinger bands must satisfy upper >= middle >= lower: %v", err)
	}
}

// Feature: security-and-performance-hardening, Property 20: 对任意收盘价序列,布林带在每个有效窗口
// 位置(i ≥ P−1)的中轨值在 eps 容差内等于同周期同位置的简单移动平均值。
func TestProperty20_BollingerMiddleEqualsSMA(t *testing.T) {
	const period = 20
	f := func(ps priceSeries) bool {
		closes := []float64(ps)
		_, mid, _ := BollingerBands(closes, period, 2.0)
		sma := SMA(closes, period)
		for i := period - 1; i < len(closes); i++ {
			if math.Abs(mid[i]-sma[i]) > eps {
				return false
			}
		}
		return true
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("Bollinger middle band must equal SMA: %v", err)
	}
}
