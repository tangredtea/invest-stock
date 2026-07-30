package backtest

import (
	"fmt"

	"invest/pkg/indicator"
)

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
func (a *PerSymbolAdapter) Params() []ParamSpec { return a.specs }
func (a *PerSymbolAdapter) MinBars() int        { return a.minBars }

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
