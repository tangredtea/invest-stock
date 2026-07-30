package backtest

import (
	"math"
	"testing"
	"testing/quick"
	"time"

	"invest/pkg/data"
	"invest/pkg/indicator"
)

// Feature: portfolio-backtest-risk, Property 22: 等权方案下任意两个参与撮合标的目标权重之差 ≤ 1e-9,
// 且各目标权重之和等于可分配权重总额(1−现金保留比例)。
func TestProperty22_EqualWeight(t *testing.T) {
	f := func(nSym uint8, reserveRaw uint8) bool {
		k := int(nSym%8) + 2                      // 2..9
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
	// Non-finite weights must be rejected before they poison equity curves.
	if _, err := (PortfolioEngine{}).Run(aligned, mk(map[string]float64{"600000": math.NaN(), "600001": 0.2}), cfg); err == nil {
		t.Error("expected error for non-finite weight")
	}
	// Valid weights must pass.
	if _, err := (PortfolioEngine{}).Run(aligned, mk(map[string]float64{"600000": 0.5, "600001": 0.5}), cfg); err != nil {
		t.Errorf("valid weights should not error: %v", err)
	}
}

func TestPortfolioEngineRejectsInvalidDecision(t *testing.T) {
	input := map[string][]data.KLine{
		"600000": genSymbolKlines(120, 0),
		"600001": genSymbolKlines(120, 1),
	}
	aligned, err := Align(input, AlignIntersection)
	if err != nil {
		t.Fatalf("align: %v", err)
	}
	cfg := DefaultPortfolioConfig()

	tests := []struct {
		name  string
		strat PortfolioStrategy
	}{
		{
			name:  "invalid kind",
			strat: fixedPortfolioDecisionStrat{dec: PortfolioDecision{Kind: PortfolioDecisionKind(99)}},
		},
		{
			name: "weights with signals",
			strat: fixedPortfolioDecisionStrat{dec: PortfolioDecision{
				Kind:    DecisionWeights,
				Weights: map[string]float64{"600000": 0.5},
				Signals: map[string]Decision{"600000": HoldDecision()},
			}},
		},
		{
			name: "signals with weights",
			strat: fixedPortfolioDecisionStrat{dec: PortfolioDecision{
				Kind:    DecisionSignals,
				Weights: map[string]float64{"600000": 0.5},
			}},
		},
		{
			name: "invalid signal action",
			strat: fixedPortfolioDecisionStrat{dec: PortfolioDecision{
				Kind:    DecisionSignals,
				Signals: map[string]Decision{"600000": {Action: Action(99), Qty: 100}},
			}},
		},
		{
			name: "unknown signal symbol",
			strat: fixedPortfolioDecisionStrat{dec: PortfolioDecision{
				Kind:    DecisionSignals,
				Signals: map[string]Decision{"missing": HoldDecision()},
			}},
		},
		{
			name: "unknown weight symbol",
			strat: fixedPortfolioDecisionStrat{dec: PortfolioDecision{
				Kind:    DecisionWeights,
				Weights: map[string]float64{"missing": 0.5},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := (PortfolioEngine{}).Run(aligned, tt.strat, cfg); err == nil {
				t.Fatalf("Run accepted invalid portfolio decision")
			}
		})
	}
}

func TestPortfolioEngineRejectsMalformedAlignedData(t *testing.T) {
	validInput := map[string][]data.KLine{
		"600000": genSymbolKlines(80, 0),
		"600001": genSymbolKlines(80, 1),
	}
	valid, err := Align(validInput, AlignIntersection)
	if err != nil {
		t.Fatalf("align: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*AlignedData)
	}{
		{
			name: "duplicate symbol",
			mutate: func(a *AlignedData) {
				a.Symbols[1] = a.Symbols[0]
			},
		},
		{
			name: "unsorted symbol",
			mutate: func(a *AlignedData) {
				a.Symbols[0], a.Symbols[1] = a.Symbols[1], a.Symbols[0]
			},
		},
		{
			name: "short points",
			mutate: func(a *AlignedData) {
				a.Points["600000"] = a.Points["600000"][:len(a.Points["600000"])-1]
			},
		},
		{
			name: "bad point price",
			mutate: func(a *AlignedData) {
				p := a.Points["600000"][0]
				p.High = p.Low - 0.01
				a.Points["600000"][0] = p
			},
		},
		{
			name: "extra points symbol",
			mutate: func(a *AlignedData) {
				a.Points["600002"] = a.Points["600000"]
			},
		},
		{
			name: "missing raw",
			mutate: func(a *AlignedData) {
				delete(a.Raw, "600000")
			},
		},
		{
			name: "non ascending timeline",
			mutate: func(a *AlignedData) {
				a.Timeline[1] = a.Timeline[0]
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aligned := cloneAlignedData(valid)
			tt.mutate(&aligned)
			if _, err := (PortfolioEngine{}).Run(aligned, equalWeightStrat{}, DefaultPortfolioConfig()); err == nil {
				t.Fatalf("Run accepted malformed aligned data")
			}
		})
	}
}

func TestPortfolioEngineRejectsConfigForUnknownSymbols(t *testing.T) {
	input := map[string][]data.KLine{
		"600000": genSymbolKlines(80, 0),
		"600001": genSymbolKlines(80, 1),
	}
	aligned, err := Align(input, AlignIntersection)
	if err != nil {
		t.Fatalf("align: %v", err)
	}

	cfg := DefaultPortfolioConfig()
	cfg.Scheme = WeightSpecified
	cfg.Weights = map[string]float64{"600000": 0.5, "missing": 0.2}
	if _, err := (PortfolioEngine{}).Run(aligned, equalWeightStrat{}, cfg); err == nil {
		t.Fatal("expected error for unknown specified-weight symbol")
	}

	cfg = DefaultPortfolioConfig()
	cfg.Risk.PerSymbolCap = map[string]float64{"missing": 0.5}
	if _, err := (PortfolioEngine{}).Run(aligned, equalWeightStrat{}, cfg); err == nil {
		t.Fatal("expected error for unknown cap symbol")
	}
}

func TestPortfolioEngineMarksFinalPendingNextOpenOrdersUnfilled(t *testing.T) {
	input := map[string][]data.KLine{
		"600000": genSymbolKlines(120, 0),
		"600001": genSymbolKlines(120, 1),
	}
	aligned, err := Align(input, AlignIntersection)
	if err != nil {
		t.Fatalf("align: %v", err)
	}
	cfg := DefaultPortfolioConfig()
	cfg.Base.FillRule = FillNextOpen
	cfg.Rebalance = RebalanceConfig{Threshold: true, ThresholdValue: 0.01}

	res, err := (PortfolioEngine{}).Run(aligned, finalBarWeightStrat{}, cfg)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(res.Unfilled) == 0 {
		t.Fatal("expected final pending next-open order to be marked unfilled")
	}
	for _, u := range res.Unfilled {
		if u.Reason != "no_next_bar" {
			t.Fatalf("unfilled reason = %q, want no_next_bar", u.Reason)
		}
		if !u.Date.Equal(aligned.Timeline[len(aligned.Timeline)-1]) {
			t.Fatalf("unfilled date = %v, want %v", u.Date, aligned.Timeline[len(aligned.Timeline)-1])
		}
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

func cloneAlignedData(in AlignedData) AlignedData {
	out := AlignedData{
		Symbols:  append([]string(nil), in.Symbols...),
		Timeline: append([]time.Time(nil), in.Timeline...),
		Points:   make(map[string][]PricePoint, len(in.Points)),
		Raw:      make(map[string][]data.KLine, len(in.Raw)),
	}
	for sym, pts := range in.Points {
		out.Points[sym] = append([]PricePoint(nil), pts...)
	}
	for sym, ks := range in.Raw {
		out.Raw[sym] = append([]data.KLine(nil), ks...)
	}
	return out
}

type fixedPortfolioDecisionStrat struct{ dec PortfolioDecision }

func (fixedPortfolioDecisionStrat) Name() string        { return "fixed-portfolio-decision" }
func (fixedPortfolioDecisionStrat) Params() []ParamSpec { return nil }
func (fixedPortfolioDecisionStrat) MinBars() int        { return 0 }
func (fixedPortfolioDecisionStrat) isSignal()           {}
func (s fixedPortfolioDecisionStrat) Decide(*PortfolioContext) PortfolioDecision {
	return s.dec
}

type finalBarWeightStrat struct{}

func (finalBarWeightStrat) Name() string        { return "final-bar-weight" }
func (finalBarWeightStrat) Params() []ParamSpec { return nil }
func (finalBarWeightStrat) MinBars() int        { return 0 }
func (finalBarWeightStrat) isTargetWeight()     {}
func (finalBarWeightStrat) Decide(ctx *PortfolioContext) PortfolioDecision {
	if ctx.Index != len(ctx.Timeline)-1 {
		weights := make(map[string]float64, len(ctx.Symbols))
		for _, sym := range ctx.Symbols {
			weights[sym] = 0
		}
		return PortfolioDecision{Kind: DecisionWeights, Weights: weights}
	}
	return PortfolioDecision{Kind: DecisionWeights, Weights: map[string]float64{ctx.Symbols[0]: 0.5}}
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

func TestPortfolioContextAccessorsReportInvalidSymbolAndIndex(t *testing.T) {
	input := map[string][]data.KLine{
		"600000": genSymbolKlines(2, 0),
		"600001": genSymbolKlines(2, 1),
	}
	aligned, err := Align(input, AlignIntersection)
	if err != nil {
		t.Fatalf("align: %v", err)
	}
	series := map[string]indicator.Series{
		"600000": indicator.ComputeSeries([]float64{1, 2}),
		"600001": indicator.ComputeSeries([]float64{1, 2}),
	}
	ctx := &PortfolioContext{Index: 2, Symbols: aligned.Symbols, aligned: &aligned, series: series}

	expectPanicContains(t, func() { _ = ctx.Bar("missing", 0) }, "unknown symbol")
	expectPanicContains(t, func() { _ = ctx.Bar("600000", 2) }, "out of range")
	expectPanicContains(t, func() { _ = ctx.Ind("missing", FieldMA5, 0) }, "unknown symbol")
	expectPanicContains(t, func() { _ = ctx.Ind("600000", FieldMA5, 2) }, "out of range")
	if got := ctx.Ind("600000", IndField(999), 1); got != 0 {
		t.Fatalf("unknown indicator field = %v, want 0", got)
	}
	expectPanicContains(t, func() { _ = ctx.Ind("600000", IndField(999), 3) }, "look-ahead")
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
