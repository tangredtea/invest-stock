package strategy

import (
	"math"
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"
)

func finite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

// t0Input is a generated, fully-valid intraday input for calcT0Prices:
// low > 0, high > 0, high >= low, price > 0, and all indicators positive
// finite reals in a realistic range. This satisfies the "valid real-time
// quote" precondition of Property 23.
type t0Input struct {
	price, ma5, ma20, rsi, bollUpper, bollLower, macdHist, prevMacdHist float64
	q                                                                  QuoteInfo
}

func (t0Input) Generate(r *rand.Rand, _ int) reflect.Value {
	pos := func() float64 { return 0.01 + r.Float64()*999.99 } // (0, 1000]
	low := pos()
	high := low + r.Float64()*100 // high >= low
	price := low + r.Float64()*(high-low)
	bollLower := pos()
	bollUpper := bollLower + r.Float64()*100
	in := t0Input{
		price:        price,
		ma5:          pos(),
		ma20:         pos(),
		rsi:          r.Float64() * 100,
		bollUpper:    bollUpper,
		bollLower:    bollLower,
		macdHist:     (r.Float64() - 0.5) * 10,
		prevMacdHist: (r.Float64() - 0.5) * 10,
		q:            QuoteInfo{Price: price, High: high, Low: low, PreClose: pos()},
	}
	return reflect.ValueOf(in)
}

func (in t0Input) call() (float64, float64) {
	return calcT0Prices(in.price, in.ma5, in.ma20, in.rsi,
		in.bollUpper, in.bollLower, in.macdHist, in.prevMacdHist, in.q)
}

// Feature: security-and-performance-hardening, Property 23: 对任意满足有效实时行情前置条件
// (最低价 > 0 ∧ 最高价 > 0 ∧ 最高价 ≥ 最低价 ∧ 现价 > 0)的输入,calcT0Prices 返回的买入价
// 严格小于卖出价(买卖价差严格大于 0)。
func TestProperty23_BuyStrictlyLessThanSell(t *testing.T) {
	f := func(in t0Input) bool {
		buy, sell := in.call()
		return buy < sell
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("with valid quote, buy price must be strictly less than sell price: %v", err)
	}
}

// Feature: security-and-performance-hardening, Property 24: 对任意使所有指标权重之和为零的输入
// (MA5 ≤ 0 ∧ MA20 ≤ 0 ∧ 并非布林带上下轨同时为正),calcT0Prices 返回买入价与卖出价均为 0。
func TestProperty24_ZeroWeightReturnsZero(t *testing.T) {
	f := func(price, rsi, macdHist, prevMacdHist float64) bool {
		// Force sumW == 0: ma5 <= 0, ma20 <= 0, and not (bollLower>0 && bollUpper>0).
		ma5 := -math.Abs(price) - 1
		ma20 := -math.Abs(rsi) - 1
		bollLower := 0.0 // disables the Bollinger contribution
		bollUpper := math.Abs(macdHist) + 1
		buy, sell := calcT0Prices(price, ma5, ma20, rsi, bollUpper, bollLower,
			macdHist, prevMacdHist, QuoteInfo{}) // no real-time data
		return buy == 0 && sell == 0
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("zero total weight must yield (0,0): %v", err)
	}
}

// Feature: security-and-performance-hardening, Property 25: 对任意取自有限实数生成域的价格与指标
// 输入,calcT0Prices 终止计算、返回有限实数(不含 NaN 与 ±Inf)的买入价与卖出价,且不引发运行时 panic。
func TestProperty25_FiniteAndNoPanic(t *testing.T) {
	f := func(price, ma5, ma20, rsi, bollUpper, bollLower, macdHist, prevMacdHist,
		qp, qh, ql, qpc float64) bool {
		buy, sell := calcT0Prices(price, ma5, ma20, rsi, bollUpper, bollLower,
			macdHist, prevMacdHist, QuoteInfo{Price: qp, High: qh, Low: ql, PreClose: qpc})
		return finite(buy) && finite(sell)
	}
	// Constrain generated floats to a finite domain via a custom values func.
	cfg := &quick.Config{
		MaxCount: 100,
		Values: func(args []reflect.Value, r *rand.Rand) {
			for i := range args {
				args[i] = reflect.ValueOf((r.Float64() - 0.5) * 2e6)
			}
		},
	}
	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("calcT0Prices must return finite values without panic: %v", err)
	}
}

// Feature: security-and-performance-hardening, Property 26: 对任意一组相同输入,calcT0Prices 多次
// 调用产生逐位相等的买入价与卖出价(计算具确定性且无副作用)。
func TestProperty26_Deterministic(t *testing.T) {
	f := func(in t0Input) bool {
		b1, s1 := in.call()
		b2, s2 := in.call()
		return b1 == b2 && s1 == s2
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("calcT0Prices must be deterministic: %v", err)
	}
}
