package indicator

import (
	"testing"
	"testing/quick"
)

// Feature: security-and-performance-hardening, Property 17: 对任意收盘价序列与回看周期 P,SMA 在每个
// 有效窗口位置(下标 i ≥ P−1)的值在 eps 容差内不小于该窗口最小收盘价、且不大于该窗口最大收盘价。
func TestProperty17_SMAWithinWindowMinMax(t *testing.T) {
	const period = 20
	f := func(ps priceSeries) bool {
		closes := []float64(ps)
		sma := SMA(closes, period)
		for i := period - 1; i < len(closes); i++ {
			lo, hi := closes[i], closes[i]
			for j := i - period + 1; j <= i; j++ {
				if closes[j] < lo {
					lo = closes[j]
				}
				if closes[j] > hi {
					hi = closes[j]
				}
			}
			if sma[i] < lo-eps || sma[i] > hi+eps {
				return false
			}
		}
		return true
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("SMA must lie within window [min,max]: %v", err)
	}
}
