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

func validatePortfolioDecision(d PortfolioDecision) error {
	switch d.Kind {
	case DecisionWeights:
		if len(d.Signals) > 0 {
			return fmt.Errorf("目标权重决策不能携带交易信号")
		}
	case DecisionSignals:
		if len(d.Weights) > 0 {
			return fmt.Errorf("交易信号决策不能携带目标权重")
		}
		for sym, sig := range d.Signals {
			if err := validateDecision(sig); err != nil {
				return fmt.Errorf("标的 %s 决策非法: %w", sym, err)
			}
		}
	default:
		return fmt.Errorf("组合策略决策类型非法: %d", d.Kind)
	}
	return nil
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
