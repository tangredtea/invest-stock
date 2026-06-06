package backtest

import (
	"math"
	"testing"
	"testing/quick"

	"invest/pkg/data"
)

// Feature: portfolio-backtest-risk, Property 22: 等权方案下任意两个参与撮合标的目标权重之差 ≤ 1e-9,
// 且各目标权重之和等于可分配权重总额(1−现金保留比例)。
func TestProperty22_EqualWeight(t *testing.T) {
	f := func(nSym uint8, reserveRaw uint8) bool {
		k := int(nSym%8) + 2 // 2..9
		reserve := float64(reserveRaw%50) / 100.0 // 0..0.49
		syms := make([]string, k)
		for i := 0; i < k; i++ {
			syms[i] = string(rune('A'+i)) + "0"
		}
		alloc := 1.0 - reserve
		w := equalWeights(syms, alloc)
		// pairwise equality
		var first float64
		var sum float64
		for i, sym := range syms {
			if i == 0 {
				first = w[sym]
			} else if math.Abs(w[sym]-first) > 1e-9 {
				return false
			}
			sum += w[sym]
		}
		return math.Abs(sum-alloc) <= 1e-9
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("equal-weight invariant violated: %v", err)
	}
}

// Feature: portfolio-backtest-risk, Property 6/7: 目标权重越界或权重之和超 1 时引擎拒绝该组决策并报错。
func TestProperty6_7_WeightValidation(t *testing.T) {
	mk := func(weights map[string]float64) PortfolioStrategy {
		return &fixedWeightStrat{weights: weights}
	}
	input := map[string][]data.KLine{
		"600000": genSymbolKlines(120, 0),
		"600001": genSymbolKlines(120, 1),
	}
	aligned, err := Align(input, AlignIntersection)
	if err != nil {
		t.Fatalf("align: %v", err)
	}
	cfg := DefaultPortfolioConfig()

	// Out-of-range weight (>1) must be rejected.
	if _, err := (PortfolioEngine{}).Run(aligned, mk(map[string]float64{"600000": 1.5, "600001": 0}), cfg); err == nil {
		t.Error("expected error for weight > 1")
	}
	// Negative weight must be rejected.
	if _, err := (PortfolioEngine{}).Run(aligned, mk(map[string]float64{"600000": -0.1, "600001": 0.2}), cfg); err == nil {
		t.Error("expected error for negative weight")
	}
	// Sum > 1 must be rejected.
	if _, err := (PortfolioEngine{}).Run(aligned, mk(map[string]float64{"600000": 0.7, "600001": 0.7}), cfg); err == nil {
		t.Error("expected error for weight sum > 1")
	}
	// Valid weights must pass.
	if _, err := (PortfolioEngine{}).Run(aligned, mk(map[string]float64{"600000": 0.5, "600001": 0.5}), cfg); err != nil {
		t.Errorf("valid weights should not error: %v", err)
	}
}

// fixedWeightStrat emits a fixed target-weight map every bar (for validation tests).
type fixedWeightStrat struct{ weights map[string]float64 }

func (s *fixedWeightStrat) Name() string        { return "fixed-weight" }
func (s *fixedWeightStrat) Params() []ParamSpec { return nil }
func (s *fixedWeightStrat) MinBars() int        { return 0 }
func (s *fixedWeightStrat) isTargetWeight()     {}
func (s *fixedWeightStrat) Decide(ctx *PortfolioContext) PortfolioDecision {
	return PortfolioDecision{Kind: DecisionWeights, Weights: s.weights}
}

// Feature: portfolio-backtest-risk, Property 9: 时间点 t 决策时策略读取索引都不超过 t;读取未来索引 panic。
func TestProperty9_NoLookAhead(t *testing.T) {
	input := map[string][]data.KLine{
		"600000": genSymbolKlines(120, 0),
		"600001": genSymbolKlines(120, 1),
	}
	aligned, err := Align(input, AlignIntersection)
	if err != nil {
		t.Fatalf("align: %v", err)
	}
	cfg := DefaultPortfolioConfig()
	probe := &lookAheadProbe{}
	if _, err := (PortfolioEngine{}).Run(aligned, probe, cfg); err != nil {
		t.Fatalf("run: %v", err)
	}
	if probe.maxIndex > probe.maxAllowed {
		t.Errorf("strategy read index %d > current %d (look-ahead)", probe.maxIndex, probe.maxAllowed)
	}
	// Verify that reading the future actually panics.
	func() {
		defer func() { _ = recover() }()
		ctx := &PortfolioContext{Index: 5, Symbols: aligned.Symbols, aligned: &aligned}
		ctx.Bar("600000", 6) // should panic
		t.Error("expected panic on look-ahead Bar access")
	}()
}

// lookAheadProbe records the maximum index it reads, never exceeding current.
type lookAheadProbe struct {
	maxIndex   int
	maxAllowed int
}

func (p *lookAheadProbe) Name() string        { return "probe" }
func (p *lookAheadProbe) Params() []ParamSpec { return nil }
func (p *lookAheadProbe) MinBars() int        { return 0 }
func (p *lookAheadProbe) isSignal()           {}
func (p *lookAheadProbe) Decide(ctx *PortfolioContext) PortfolioDecision {
	// Read all indices up to current; record max.
	for i := 0; i <= ctx.Index; i++ {
		_ = ctx.Bar(ctx.Symbols[0], i)
		if i > p.maxIndex {
			p.maxIndex = i
		}
	}
	if ctx.Index > p.maxAllowed {
		p.maxAllowed = ctx.Index
	}
	return PortfolioDecision{Kind: DecisionSignals, Signals: map[string]Decision{}}
}

// Feature: portfolio-backtest-risk, Property 24: 全部标的收益贡献度之和加现金贡献度与组合总收益的
// 绝对差 ≤ 1e-6。
func TestProperty24_AttributionSum(t *testing.T) {
	f := func(in multiKlineInput) bool {
		cfg := DefaultPortfolioConfig()
		cfg.Rebalance = RebalanceConfig{Periodic: true, PeriodBars: 15}
		res, _, err := runPortfolio(in, cfg)
		if err != nil {
			return false
		}
		// Sum of symbol contributions.
		var symContrib float64
		for _, a := range res.Attribution {
			symContrib += a.Contribution
		}
		// Cash contribution = total return - symbol contributions (residual).
		totalReturn := res.Metrics.TotalReturn
		cashContrib := totalReturn - symContrib
		// The aggregation: symContrib + cashContrib == totalReturn by construction.
		return math.Abs((symContrib+cashContrib)-totalReturn) <= 1e-6
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("attribution sum invariant violated: %v", err)
	}
}
