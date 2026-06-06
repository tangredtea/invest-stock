package backtest

import (
	"math"
	"testing"
	"time"

	"invest/pkg/data"
)

// equalWeightStrat is a simple target-weight portfolio strategy that requests
// equal weights across all tradable symbols each bar.
type equalWeightStrat struct{}

func (equalWeightStrat) Name() string         { return "equal-weight" }
func (equalWeightStrat) Params() []ParamSpec  { return nil }
func (equalWeightStrat) MinBars() int         { return 0 }
func (equalWeightStrat) isTargetWeight()      {}
func (equalWeightStrat) Decide(ctx *PortfolioContext) PortfolioDecision {
	w := make(map[string]float64)
	tradable := 0
	for _, sym := range ctx.Symbols {
		if ctx.CanTrade(sym) {
			tradable++
		}
	}
	if tradable > 0 {
		each := 1.0 / float64(tradable)
		for _, sym := range ctx.Symbols {
			if ctx.CanTrade(sym) {
				w[sym] = each
			}
		}
	}
	return PortfolioDecision{Kind: DecisionWeights, Weights: w}
}

// genSymbolKlines builds a deterministic kline series with a phase offset.
func genSymbolKlines(n int, phase float64) []data.KLine {
	out := make([]data.KLine, n)
	base := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)
	for i := 0; i < n; i++ {
		price := 10 + 3*math.Sin(float64(i)/9+phase)
		out[i] = data.KLine{
			Date: base.AddDate(0, 0, i), Open: price, High: price + 0.2,
			Low: price - 0.2, Close: price, Volume: 1000,
		}
	}
	return out
}

func TestPortfolioEngineSmokeTargetWeight(t *testing.T) {
	input := map[string][]data.KLine{
		"600000": genSymbolKlines(200, 0),
		"600001": genSymbolKlines(200, 1.5),
		"600002": genSymbolKlines(200, 3.0),
	}
	aligned, err := Align(input, AlignIntersection)
	if err != nil {
		t.Fatalf("align: %v", err)
	}
	cfg := DefaultPortfolioConfig()
	cfg.Rebalance = RebalanceConfig{Periodic: true, PeriodBars: 20}

	res, err := PortfolioEngine{}.Run(aligned, equalWeightStrat{}, cfg)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	n := len(aligned.Timeline)
	if len(res.Equity) != n || len(res.Drawdown) != n || len(res.NetValue) != n {
		t.Errorf("curve length mismatch: got %d/%d/%d, want %d", len(res.Equity), len(res.Drawdown), len(res.NetValue), n)
	}
	if math.Abs(res.NetValue[0]-1) > 1e-9 {
		t.Errorf("netValue[0] = %v, want 1", res.NetValue[0])
	}
	if math.Abs(res.Equity[0]-cfg.Base.InitialCash) > 1e-6 {
		t.Errorf("equity[0] = %v, want %v", res.Equity[0], cfg.Base.InitialCash)
	}
	if res.Metrics.MaxDrawdown < 0 || res.Metrics.MaxDrawdown > 1 {
		t.Errorf("maxDrawdown = %v out of [0,1]", res.Metrics.MaxDrawdown)
	}
	for _, sym := range aligned.Symbols {
		if len(res.Weights[sym]) != n {
			t.Errorf("weights[%s] length = %d, want %d", sym, len(res.Weights[sym]), n)
		}
	}
	if len(res.Attribution) != len(aligned.Symbols) {
		t.Errorf("attribution count = %d, want %d", len(res.Attribution), len(aligned.Symbols))
	}
	t.Logf("rebalances=%d finalEquity=%.2f maxDD=%.4f sharpe=%.2f", res.RebalanceCount, res.Equity[n-1], res.Metrics.MaxDrawdown, res.Metrics.Sharpe)
}

func TestPortfolioEnginePerSymbolAdapter(t *testing.T) {
	input := map[string][]data.KLine{
		"600000": genSymbolKlines(200, 0),
		"600001": genSymbolKlines(200, 2.0),
	}
	aligned, err := Align(input, AlignIntersection)
	if err != nil {
		t.Fatalf("align: %v", err)
	}
	// Register a simple single-symbol strategy in a local registry (the backtest
	// package cannot import builtin without a cycle).
	reg := NewRegistry()
	_ = reg.Register("ma-cross", []ParamSpec{
		{Name: "startIndex", Type: ParamInt, Default: IntVal(60), Min: 0, Max: 100000},
	}, func(p map[string]ParamValue) (Strategy, error) {
		return &smokeMACross{startIndex: p["startIndex"].Int}, nil
	})
	adapter, err := NewPerSymbolAdapter(reg, "ma-cross", nil, aligned.Symbols)
	if err != nil {
		t.Fatalf("adapter: %v", err)
	}
	cfg := DefaultPortfolioConfig()
	res, err := PortfolioEngine{}.Run(aligned, adapter, cfg)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(res.Equity) != len(aligned.Timeline) {
		t.Errorf("equity length mismatch")
	}
	for tIdx := 0; tIdx < len(res.Equity); tIdx++ {
		if math.IsNaN(res.Equity[tIdx]) || math.IsInf(res.Equity[tIdx], 0) {
			t.Fatalf("equity[%d] not finite", tIdx)
		}
	}
}

// smokeMACross is a minimal single-symbol MA-cross strategy for adapter tests.
type smokeMACross struct{ startIndex int }

func (s *smokeMACross) Name() string        { return "ma-cross" }
func (s *smokeMACross) Params() []ParamSpec { return nil }
func (s *smokeMACross) MinBars() int        { return s.startIndex + 1 }
func (s *smokeMACross) Decide(ctx *Context) Decision {
	i := ctx.Index
	if i < s.startIndex+1 {
		return HoldDecision()
	}
	prevAbove := ctx.Ind(FieldMA5, i-1) > ctx.Ind(FieldMA20, i-1)
	currAbove := ctx.Ind(FieldMA5, i) > ctx.Ind(FieldMA20, i)
	if !prevAbove && currAbove && ctx.Shares == 0 {
		return BuyAmount(ctx.Cash)
	}
	if prevAbove && !currAbove && ctx.Shares > 0 {
		return SellQty(ctx.Shares)
	}
	return HoldDecision()
}
