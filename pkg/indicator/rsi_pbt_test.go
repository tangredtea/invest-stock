package indicator

import (
	"testing"
	"testing/quick"
)

// Feature: security-and-performance-hardening, Property 18: 对任意收盘价序列,RSI 的每个输出值都处于
// 闭区间 [0, 100] 内(含端点,eps 容差)。
func TestProperty18_RSIWithinZeroHundred(t *testing.T) {
	f := func(ps priceSeries) bool {
		rsi := RSI([]float64(ps), 14)
		for _, v := range rsi {
			if v < 0-eps || v > 100+eps {
				return false
			}
		}
		return true
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("RSI must stay within [0,100]: %v", err)
	}
}
