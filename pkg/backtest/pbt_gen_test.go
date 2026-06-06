package backtest

import (
	"math"
	"math/rand"
	"reflect"
	"time"

	"invest/pkg/data"
)

const eps = 1e-9

// klineSeq is a generated, valid (strictly ascending dates) K线 series with
// length >= 60 so it can drive the engine. Prices are finite reals in a
// realistic range.
type klineSeq []data.KLine

func (klineSeq) Generate(r *rand.Rand, _ int) reflect.Value {
	n := 60 + r.Intn(200) // [60, 259]
	base := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)
	out := make([]data.KLine, n)
	price := 5 + r.Float64()*50
	for i := 0; i < n; i++ {
		// Random walk constrained to [0.5, +inf), mild moves.
		price += (r.Float64() - 0.5) * 2
		if price < 0.5 {
			price = 0.5
		}
		hi := price + r.Float64()*1.5
		lo := price - r.Float64()*1.5
		if lo < 0.1 {
			lo = 0.1
		}
		out[i] = data.KLine{
			Date:   base.AddDate(0, 0, i),
			Open:   lo + r.Float64()*(hi-lo),
			High:   hi,
			Low:    lo,
			Close:  price,
			Volume: 1000 + r.Float64()*1e6,
		}
	}
	return reflect.ValueOf(klineSeq(out))
}

func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

// testStrategy is a simple parametric strategy used to exercise the engine in
// property tests: it buys a fixed amount on the first eligible bar, then holds.
type testStrategy struct {
	startIndex int
	buyAmount  float64
}

func (s *testStrategy) Name() string { return "test-buy-hold" }
func (s *testStrategy) MinBars() int { return s.startIndex + 1 }
func (s *testStrategy) Params() []ParamSpec {
	return []ParamSpec{
		{Name: "startIndex", Type: ParamInt, Default: IntVal(60), Min: 0, Max: 100000},
	}
}
func (s *testStrategy) Decide(ctx *Context) Decision {
	if ctx.Index == s.startIndex && ctx.Shares == 0 {
		return BuyAmount(s.buyAmount)
	}
	return HoldDecision()
}
