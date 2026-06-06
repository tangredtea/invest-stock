package builtin

import (
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"
	"time"

	"invest/pkg/backtest"
	"invest/pkg/data"
	"invest/pkg/indicator"
)

// ctxGen builds a valid decision Context at a random index within a random
// kline series, with aligned indicator series.
type ctxInput struct {
	klines []data.KLine
	index  int
}

func (ctxInput) Generate(r *rand.Rand, _ int) reflect.Value {
	n := 80 + r.Intn(120)
	base := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)
	out := make([]data.KLine, n)
	price := 10 + r.Float64()*20
	for i := 0; i < n; i++ {
		price += (r.Float64() - 0.5) * 1.5
		if price < 0.5 {
			price = 0.5
		}
		hi := price + r.Float64()
		lo := price - r.Float64()
		if lo < 0.1 {
			lo = 0.1
		}
		out[i] = data.KLine{Date: base.AddDate(0, 0, i), Open: price, High: hi, Low: lo, Close: price, Volume: 1000}
	}
	idx := 60 + r.Intn(n-60)
	return reflect.ValueOf(ctxInput{klines: out, index: idx})
}

func buildContext(in ctxInput) *backtest.Context {
	closes := make([]float64, len(in.klines))
	for i, k := range in.klines {
		closes[i] = k.Close
	}
	return &backtest.Context{
		Index: in.index, Klines: in.klines,
		Series: indicator.ComputeSeries(closes),
		Cash:   100000, Shares: 100,
	}
}

func allStrategies(t *testing.T) []backtest.Strategy {
	t.Helper()
	var out []backtest.Strategy
	for _, name := range Names() {
		s, err := backtest.Default.New(name, nil)
		if err != nil {
			t.Fatalf("New(%q): %v", name, err)
		}
		out = append(out, s)
	}
	return out
}

// Feature: quant-backtest-platform, Property 1: 对任意内置策略实例与任意合法决策上下文,Decide 返回的
// Decision 的 Action ∈ {Hold,Buy,Sell};Hold 时 Qty==0 且 Amount==0;Buy/Sell 时恰有 Qty>0 与
// Amount>0 之一成立。
func TestProperty1_DecisionWellFormed(t *testing.T) {
	strategies := allStrategies(t)
	f := func(in ctxInput) bool {
		ctx := buildContext(in)
		for _, s := range strategies {
			d := s.Decide(ctx)
			switch d.Action {
			case backtest.Hold:
				if d.Qty != 0 || d.Amount != 0 {
					return false
				}
			case backtest.Buy, backtest.Sell:
				qtyPos := d.Qty > 0
				amtPos := d.Amount > 0
				if qtyPos == amtPos { // must be exactly one (XOR)
					return false
				}
			default:
				return false
			}
		}
		return true
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("decision well-formedness violated: %v", err)
	}
}

// Feature: quant-backtest-platform, Property 2: 对任意 K线序列、当前索引 i,以及任意仅作用于索引 j>i
// 的数据扰动,内置策略在第 i 根的决策必与扰动前一致。
func TestProperty2_NoLookAhead(t *testing.T) {
	f := func(in ctxInput) bool {
		// Build two series identical up to index, differing only after it.
		orig := buildContext(in)
		perturbed := make([]data.KLine, len(in.klines))
		copy(perturbed, in.klines)
		for j := in.index + 1; j < len(perturbed); j++ {
			perturbed[j].Close *= 1.5
			perturbed[j].High *= 1.5
			perturbed[j].Low *= 0.5
			perturbed[j].Open *= 1.3
		}
		closes := make([]float64, len(perturbed))
		for i, k := range perturbed {
			closes[i] = k.Close
		}
		pctx := &backtest.Context{
			Index: in.index, Klines: perturbed,
			Series: indicator.ComputeSeries(closes), Cash: 100000, Shares: 100,
		}
		// Fresh strategy instances each time (some are stateful across bars; here
		// we only compare a single-bar decision at the same index).
		for _, name := range Names() {
			s1, _ := backtest.Default.New(name, nil)
			s2, _ := backtest.Default.New(name, nil)
			d1 := s1.Decide(orig)
			d2 := s2.Decide(pctx)
			if d1 != d2 {
				return false
			}
		}
		return true
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("look-ahead detected: %v", err)
	}
}

// Feature: quant-backtest-platform, Property 5: 对任意已注册策略,其 Params() 的每个 ParamSpec 满足:
// 名称非空且唯一;Type 合法;数值型 Min<=Max 且 Default∈[Min,Max];枚举型 Default∈Enum。
func TestProperty5_ParamSpecSelfConsistent(t *testing.T) {
	strategies := allStrategies(t)
	for _, s := range strategies {
		seen := map[string]bool{}
		for _, p := range s.Params() {
			if p.Name == "" || seen[p.Name] {
				t.Errorf("%s: param name empty or duplicate: %q", s.Name(), p.Name)
			}
			seen[p.Name] = true
			switch p.Type {
			case backtest.ParamInt:
				if p.Min > p.Max || float64(p.Default.Int) < p.Min || float64(p.Default.Int) > p.Max {
					t.Errorf("%s.%s: int default out of [min,max]", s.Name(), p.Name)
				}
			case backtest.ParamFloat:
				if p.Min > p.Max || p.Default.Float < p.Min || p.Default.Float > p.Max {
					t.Errorf("%s.%s: float default out of [min,max]", s.Name(), p.Name)
				}
			case backtest.ParamEnum:
				ok := false
				for _, e := range p.Enum {
					if e == p.Default.Str {
						ok = true
					}
				}
				if !ok {
					t.Errorf("%s.%s: enum default not in enum set", s.Name(), p.Name)
				}
			case backtest.ParamBool:
				// any bool valid
			default:
				t.Errorf("%s.%s: invalid param type", s.Name(), p.Name)
			}
		}
	}
}
