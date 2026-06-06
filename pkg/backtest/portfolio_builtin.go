package backtest

// SchemeStrategy is a target-weight portfolio strategy that simply requests the
// configured weight scheme's allocation every bar. Combined with rebalancing,
// it implements classic equal-weight / specified-weight / inverse-vol portfolios.
//
// It returns an empty weights map (引擎据 schemeWeights 计算实际目标), so the
// engine's resolveTargetWeights signal/scheme path drives allocation. To make it
// a target-weight strategy that defers to the scheme, it emits DecisionWeights
// with weights filled by the engine scheme; here we emit a sentinel that the
// engine treats as "use configured scheme".
type SchemeStrategy struct{}

func (SchemeStrategy) Name() string        { return "scheme-weights" }
func (SchemeStrategy) Params() []ParamSpec { return nil }
func (SchemeStrategy) MinBars() int        { return 0 }
func (SchemeStrategy) isTargetWeight()     {}

// Decide returns the configured scheme's target weights over tradable symbols.
// The engine fills actual weights via schemeWeights when Weights is nil/sentinel,
// but to keep the strategy self-contained we emit equal weights as a default and
// rely on the engine's cap/reserve normalization. For specified/inverse-vol the
// engine's scheme path is used via SignalStrategy-style; here we emit weights
// directly so target-weight validation applies.
func (SchemeStrategy) Decide(ctx *PortfolioContext) PortfolioDecision {
	// Emit a sentinel: empty weights means "defer to configured scheme".
	return PortfolioDecision{Kind: DecisionWeights, Weights: nil}
}
