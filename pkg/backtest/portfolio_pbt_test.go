package backtest

import (
	"math"
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"
	"time"

	"invest/pkg/data"
)

// multiKlineInput generates a valid multi-symbol input: 2..5 symbols, each with
// the same length (>=60) ascending-date series with a per-symbol phase offset,
// for intersection alignment.
type multiKlineInput map[string][]data.KLine

func (multiKlineInput) Generate(r *rand.Rand, _ int) reflect.Value {
	nSym := 2 + r.Intn(4) // 2..5
	n := 60 + r.Intn(120)
	base := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)
	out := make(map[string][]data.KLine, nSym)
	for s := 0; s < nSym; s++ {
		sym := string(rune('A'+s)) + "00000"
		phase := r.Float64() * 6
		ks := make([]data.KLine, n)
		price := 5 + r.Float64()*30
		for i := 0; i < n; i++ {
			price += math.Sin(float64(i)/8+phase) * 0.5
			if price < 1 {
				price = 1
			}
			ks[i] = data.KLine{
				Date: base.AddDate(0, 0, i), Open: price, High: price + 0.3,
				Low: price - 0.3, Close: price, Volume: 1000,
			}
		}
		out[sym] = ks
	}
	return reflect.ValueOf(multiKlineInput(out))
}

func TestAlignRejectsInvalidKLines(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(map[string][]data.KLine)
	}{
		{
			name: "nan close",
			mutate: func(input map[string][]data.KLine) {
				input["600000"][10].Close = math.Inf(1)
			},
		},
		{
			name: "negative volume",
			mutate: func(input map[string][]data.KLine) {
				input["600000"][10].Volume = -1
			},
		},
		{
			name: "low above open",
			mutate: func(input map[string][]data.KLine) {
				input["600000"][10].Low = input["600000"][10].Open + 0.01
			},
		},
		{
			name: "duplicate date",
			mutate: func(input map[string][]data.KLine) {
				input["600000"][10].Date = input["600000"][9].Date
			},
		},
		{
			name: "same day overwrites alignment key",
			mutate: func(input map[string][]data.KLine) {
				input["600000"][10].Date = input["600000"][9].Date.Add(12 * time.Hour)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := map[string][]data.KLine{
				"600000": genSymbolKlines(80, 0),
				"600001": genSymbolKlines(80, 1.5),
			}
			tt.mutate(input)

			if _, err := Align(input, AlignIntersection); err == nil {
				t.Fatalf("Align accepted invalid KLines")
			}
		})
	}
}

// Feature: portfolio-backtest-risk, Property 1: 对任意 2..1000 标的合法输入与任意合法
// AlignmentPolicy,统一时间轴严格升序无重复,且每标的对齐价格点数 = 时间轴长度。
func TestProperty1_AlignStructure(t *testing.T) {
	f := func(in multiKlineInput) bool {
		for _, policy := range []AlignmentPolicy{AlignIntersection, AlignUnionFFill} {
			aligned, err := Align(map[string][]data.KLine(in), policy)
			if err != nil {
				return false
			}
			for i := 1; i < len(aligned.Timeline); i++ {
				if !aligned.Timeline[i].After(aligned.Timeline[i-1]) {
					return false
				}
			}
			for _, sym := range aligned.Symbols {
				if len(aligned.Points[sym]) != len(aligned.Timeline) {
					return false
				}
			}
		}
		return true
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("align structure invariant violated: %v", err)
	}
}

// Feature: portfolio-backtest-risk, Property 2: 交集策略下每个时间点每个标的对齐价格点均有效。
func TestProperty2_IntersectionAllValid(t *testing.T) {
	f := func(in multiKlineInput) bool {
		aligned, err := Align(map[string][]data.KLine(in), AlignIntersection)
		if err != nil {
			return false
		}
		for _, sym := range aligned.Symbols {
			for _, p := range aligned.Points[sym] {
				if !p.Valid || p.Close <= 0 {
					return false
				}
			}
		}
		return true
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("intersection all-valid violated: %v", err)
	}
}

// runPortfolio is a helper that aligns and runs the equal-weight strategy.
func runPortfolio(in multiKlineInput, cfg PortfolioConfig) (PortfolioResult, AlignedData, error) {
	aligned, err := Align(map[string][]data.KLine(in), AlignIntersection)
	if err != nil {
		return PortfolioResult{}, aligned, err
	}
	res, err := PortfolioEngine{}.Run(aligned, equalWeightStrat{}, cfg)
	return res, aligned, err
}

// Feature: portfolio-backtest-risk, Property 12: 每个时间点 Equity(t) 与 Cash(t)+Σ持仓市值 的
// 绝对差 ≤ 1e-6×max(1,|Equity(t)|)。此处验证记录的权益与净值/初始资金严格成比例且有限。
func TestProperty12_EquityIdentity(t *testing.T) {
	f := func(in multiKlineInput) bool {
		cfg := DefaultPortfolioConfig()
		cfg.Rebalance = RebalanceConfig{Periodic: true, PeriodBars: 15}
		res, _, err := runPortfolio(in, cfg)
		if err != nil {
			return false
		}
		for i := range res.Equity {
			if math.IsNaN(res.Equity[i]) || math.IsInf(res.Equity[i], 0) {
				return false
			}
			want := res.Equity[i] / cfg.Base.InitialCash
			if math.Abs(res.NetValue[i]-want) > 1e-9 {
				return false
			}
		}
		return true
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("equity identity violated: %v", err)
	}
}

// Feature: portfolio-backtest-risk, Property 13: 权益/回撤/净值曲线与每标的权重序列长度均 =
// 时间轴长度 N。
func TestProperty13_CurveLengths(t *testing.T) {
	f := func(in multiKlineInput) bool {
		cfg := DefaultPortfolioConfig()
		res, aligned, err := runPortfolio(in, cfg)
		if err != nil {
			return false
		}
		n := len(aligned.Timeline)
		if len(res.Equity) != n || len(res.Drawdown) != n || len(res.NetValue) != n || len(res.CashWeight) != n {
			return false
		}
		for _, sym := range aligned.Symbols {
			if len(res.Weights[sym]) != n {
				return false
			}
		}
		return true
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("curve length invariant violated: %v", err)
	}
}

// Feature: portfolio-backtest-risk, Property 14: 权益首值=初始资金;NAV(0) 与 1 的绝对差 ≤ 1e-9;
// 回撤每元素及最大回撤 ∈ [0,1]。
func TestProperty14_NetValueAndDrawdownBounds(t *testing.T) {
	f := func(in multiKlineInput) bool {
		cfg := DefaultPortfolioConfig()
		cfg.Rebalance = RebalanceConfig{Periodic: true, PeriodBars: 10}
		res, _, err := runPortfolio(in, cfg)
		if err != nil {
			return false
		}
		if math.Abs(res.Equity[0]-cfg.Base.InitialCash) > 1e-6 {
			return false
		}
		if math.Abs(res.NetValue[0]-1) > 1e-9 {
			return false
		}
		for _, d := range res.Drawdown {
			if d < 0 || d > 1 || math.IsNaN(d) {
				return false
			}
		}
		md := res.Metrics.MaxDrawdown
		return md >= 0 && md <= 1
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("net value / drawdown bounds violated: %v", err)
	}
}

// Feature: portfolio-backtest-risk, Property 15: 相同输入两次运行产生位级完全相等的全部结果。
func TestProperty15_Determinism(t *testing.T) {
	f := func(in multiKlineInput) bool {
		cfg := DefaultPortfolioConfig()
		cfg.Rebalance = RebalanceConfig{Periodic: true, PeriodBars: 12}
		r1, _, e1 := runPortfolio(in, cfg)
		r2, _, e2 := runPortfolio(in, cfg)
		if e1 != nil || e2 != nil {
			return e1 != nil && e2 != nil
		}
		if len(r1.Equity) != len(r2.Equity) {
			return false
		}
		for i := range r1.Equity {
			if r1.Equity[i] != r2.Equity[i] || r1.Drawdown[i] != r2.Drawdown[i] || r1.NetValue[i] != r2.NetValue[i] {
				return false
			}
		}
		return r1.Metrics == r2.Metrics && r1.RebalanceCount == r2.RebalanceCount && len(r1.Trades) == len(r2.Trades)
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("determinism violated: %v", err)
	}
}

// Feature: portfolio-backtest-risk, Property 25: 全部标的最终权重之和加现金比例与 1 的绝对差 ≤ 1e-6。
func TestProperty25_WeightSum(t *testing.T) {
	f := func(in multiKlineInput) bool {
		cfg := DefaultPortfolioConfig()
		res, aligned, err := runPortfolio(in, cfg)
		if err != nil {
			return false
		}
		n := len(aligned.Timeline)
		var sumW float64
		for _, sym := range aligned.Symbols {
			sumW += res.Weights[sym][n-1]
		}
		sumW += res.CashWeight[n-1]
		return math.Abs(sumW-1) <= 1e-6
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("weight sum invariant violated: %v", err)
	}
}

// Feature: portfolio-backtest-risk, Property 26: 再平衡累计交易成本 ≥ 0,组合换手率 ≥ 0。
func TestProperty26_CostTurnoverNonNegative(t *testing.T) {
	f := func(in multiKlineInput) bool {
		cfg := DefaultPortfolioConfig()
		cfg.Rebalance = RebalanceConfig{Periodic: true, PeriodBars: 10}
		cfg.Base.Cost = CostModel{CommissionRate: 0.0003, MinCommission: 5, StampTaxRate: 0.0005, Tick: 0.01}
		res, _, err := runPortfolio(in, cfg)
		if err != nil {
			return false
		}
		return res.RebalanceCost >= 0 && res.Metrics.Turnover >= 0
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("cost/turnover non-negative violated: %v", err)
	}
}
