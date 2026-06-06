package backtest

import (
	"fmt"
	"time"

	"invest/pkg/indicator"
)

// PortfolioDecisionKind distinguishes the two strategy expression styles.
type PortfolioDecisionKind int

const (
	DecisionWeights PortfolioDecisionKind = iota // target-weight style
	DecisionSignals                              // signal style
)

// PortfolioDecision is a portfolio strategy's output at one timeline point
// (Requirements 2.2, 2.3, 2.4).
type PortfolioDecision struct {
	Kind    PortfolioDecisionKind
	Weights map[string]float64  // sym -> target weight in [0,1]  (DecisionWeights)
	Signals map[string]Decision // sym -> reused single-symbol Decision (DecisionSignals)
}

// PortfolioContext is the read-only view the engine provides to a portfolio
// strategy at timeline point Index. Reads at index > Index panic (Requirement 2.7).
type PortfolioContext struct {
	Index    int
	Timeline []time.Time
	Symbols  []string
	aligned  *AlignedData
	series   map[string]indicator.Series
	Cash     float64
	Shares   map[string]int
	Equity   float64
}

// CanTrade reports whether a symbol is tradable at the current point
// (not suspended, not missing) (Requirements 1.5, 2.2).
func (c *PortfolioContext) CanTrade(sym string) bool {
	pts, ok := c.aligned.Points[sym]
	if !ok || c.Index < 0 || c.Index >= len(pts) {
		return false
	}
	return pts[c.Index].Valid
}

// Bar returns a symbol's aligned price point at index i; panics if i > Index
// (look-ahead protection, Requirement 2.7).
func (c *PortfolioContext) Bar(sym string, i int) PricePoint {
	if i < 0 || i > c.Index {
		panic(fmt.Sprintf("portfolio: look-ahead access Bar(%s,%d) at index %d", sym, i, c.Index))
	}
	return c.aligned.Points[sym][i]
}

// Ind returns a symbol's indicator field at index i; panics if i > Index.
func (c *PortfolioContext) Ind(sym string, field IndField, i int) float64 {
	if i < 0 || i > c.Index {
		panic(fmt.Sprintf("portfolio: look-ahead access Ind(%s,%d,%d) at index %d", sym, field, i, c.Index))
	}
	return indFieldAt(c.series[sym], field, i)
}

// ValidBars returns the number of valid (non-suspended, non-missing) price
// points for a symbol up to and including the current index (Requirement 2.11).
func (c *PortfolioContext) ValidBars(sym string) int {
	pts := c.aligned.Points[sym]
	n := 0
	for i := 0; i <= c.Index && i < len(pts); i++ {
		if pts[i].Valid {
			n++
		}
	}
	return n
}

// indFieldAt extracts an indicator field value at index i from a Series.
func indFieldAt(s indicator.Series, field IndField, i int) float64 {
	switch field {
	case FieldMA5:
		return s.MA5[i]
	case FieldMA20:
		return s.MA20[i]
	case FieldMA60:
		return s.MA60[i]
	case FieldRSI14:
		return s.RSI14[i]
	case FieldBollUpper:
		return s.BollUpper[i]
	case FieldBollMid:
		return s.BollMid[i]
	case FieldBollLower:
		return s.BollLower[i]
	case FieldMACDLine:
		return s.MACDLine[i]
	case FieldMACDSignal:
		return s.MACDSignal[i]
	case FieldMACDHist:
		return s.MACDHist[i]
	default:
		return 0
	}
}

// PortfolioStrategy is the unified portfolio-level strategy interface (Requirement 2.1).
type PortfolioStrategy interface {
	Name() string
	Params() []ParamSpec
	MinBars() int
	Decide(ctx *PortfolioContext) PortfolioDecision
}

// TargetWeightStrategy and SignalStrategy are marker interfaces to let the
// engine and tests distinguish expression styles; both satisfy PortfolioStrategy.
type TargetWeightStrategy interface {
	PortfolioStrategy
	isTargetWeight()
}

type SignalStrategy interface {
	PortfolioStrategy
	isSignal()
}

// PerSymbolAdapter runs an existing single-symbol Strategy independently for
// each symbol, with independent state, so its per-bar decisions are bit-for-bit
// identical to running that strategy on the single-symbol engine
// (Requirements 2.6, 2.11, 8.5).
type PerSymbolAdapter struct {
	name      string
	minBars   int
	specs     []ParamSpec
	instances map[string]Strategy
}

func (a *PerSymbolAdapter) isSignal() {}

// NewPerSymbolAdapter builds, via the registry, one independent same-name
// same-param strategy instance per symbol (Requirement 2.6).
func NewPerSymbolAdapter(reg *Registry, name string, params map[string]ParamValue, symbols []string) (*PerSymbolAdapter, error) {
	info, err := reg.Info(name)
	if err != nil {
		return nil, err
	}
	instances := make(map[string]Strategy, len(symbols))
	minBars := 0
	for _, sym := range symbols {
		s, err := reg.New(name, params)
		if err != nil {
			return nil, fmt.Errorf("为标的 %s 构造策略 %q 失败: %w", sym, name, err)
		}
		instances[sym] = s
		if s.MinBars() > minBars {
			minBars = s.MinBars()
		}
	}
	return &PerSymbolAdapter{name: name, minBars: minBars, specs: info.Params, instances: instances}, nil
}

func (a *PerSymbolAdapter) Name() string        { return a.name }
func (a *PerSymbolAdapter) Params() []ParamSpec  { return a.specs }
func (a *PerSymbolAdapter) MinBars() int         { return a.minBars }

// Decide runs each symbol's independent strategy on a single-symbol Context
// built from that symbol's valid-bar window, matching the single-symbol engine
// exactly (Requirements 2.6, 2.10, 2.11, 8.5).
func (a *PerSymbolAdapter) Decide(ctx *PortfolioContext) PortfolioDecision {
	sig := make(map[string]Decision, len(ctx.Symbols))
	for _, sym := range ctx.Symbols {
		if !ctx.CanTrade(sym) {
			continue // suspended/missing: no decision (Requirement 2.2)
		}
		inst := a.instances[sym]
		single := singleSymbolContext(ctx, sym)
		if single == nil || single.Index < inst.MinBars()-1 {
			sig[sym] = HoldDecision() // insufficient bars (Requirement 2.11)
			continue
		}
		sig[sym] = inst.Decide(single)
	}
	return PortfolioDecision{Kind: DecisionSignals, Signals: sig}
}

// singleSymbolContext assembles a single-symbol *Context for sym from the
// portfolio context's valid-bar window, so the wrapped strategy sees exactly
// what it would in the single-symbol engine (Requirement 8.5).
func singleSymbolContext(ctx *PortfolioContext, sym string) *Context {
	klines := ctx.aligned.Raw[sym]
	// The portfolio context aligns on a unified timeline; for bit-for-bit
	// parity with the single-symbol engine, the wrapped strategy operates on
	// the symbol's own raw klines indexed by how many valid bars have elapsed.
	vb := ctx.ValidBars(sym)
	if vb == 0 {
		return nil
	}
	idx := vb - 1
	if idx >= len(klines) {
		idx = len(klines) - 1
	}
	closes := make([]float64, len(klines))
	for i, k := range klines {
		closes[i] = k.Close
	}
	series := indicator.ComputeSeries(closes)
	return &Context{
		Index:  idx,
		Klines: klines,
		Series: series,
		Cash:   ctx.Cash,
		Shares: ctx.Shares[sym],
	}
}
