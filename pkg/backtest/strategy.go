package backtest

import (
	"fmt"
	"math"
)

// Action is a decision direction; exactly three values (Requirement 1.2).
type Action int

const (
	Hold Action = iota
	Buy
	Sell
)

func (a Action) String() string {
	switch a {
	case Hold:
		return "hold"
	case Buy:
		return "buy"
	case Sell:
		return "sell"
	default:
		return fmt.Sprintf("unknown(%d)", a)
	}
}

// Decision is a strategy's output on a single bar (Requirements 1.2, 1.3, 1.4).
// Semantics:
//   - Hold     => Qty==0 && Amount==0
//   - Buy/Sell => exactly one of Qty>0 (shares) or Amount>0 (yuan)
type Decision struct {
	Action Action
	Qty    int     // shares, > 0 only when ordering by quantity
	Amount float64 // yuan, > 0 only when ordering by amount
}

// HoldDecision returns a no-op decision.
func HoldDecision() Decision { return Decision{Action: Hold} }

// BuyQty / BuyAmount / SellQty / SellAmount build well-formed buy/sell decisions.
func BuyQty(qty int) Decision            { return Decision{Action: Buy, Qty: qty} }
func BuyAmount(amount float64) Decision  { return Decision{Action: Buy, Amount: amount} }
func SellQty(qty int) Decision           { return Decision{Action: Sell, Qty: qty} }
func SellAmount(amount float64) Decision { return Decision{Action: Sell, Amount: amount} }

func validateDecision(d Decision) error {
	if d.Action != Hold && d.Action != Buy && d.Action != Sell {
		return fmt.Errorf("策略决策动作非法: %d", d.Action)
	}
	if d.Qty < 0 {
		return fmt.Errorf("策略决策数量不能为负: %d", d.Qty)
	}
	if math.IsNaN(d.Amount) || math.IsInf(d.Amount, 0) || d.Amount < 0 {
		return fmt.Errorf("策略决策金额非法: %g", d.Amount)
	}
	if d.Action == Hold {
		if d.Qty != 0 || d.Amount != 0 {
			return fmt.Errorf("持有决策不能携带数量或金额")
		}
		return nil
	}
	if d.Qty > 0 && d.Amount > 0 {
		return fmt.Errorf("买卖决策不能同时按数量和金额下单")
	}
	if d.Qty == 0 && d.Amount == 0 {
		return fmt.Errorf("买卖决策必须提供数量或金额")
	}
	return nil
}

// IndField identifies an indicator series in the Context (look-ahead-safe access).
type IndField int

const (
	FieldMA5 IndField = iota
	FieldMA20
	FieldMA60
	FieldRSI14
	FieldBollUpper
	FieldBollMid
	FieldBollLower
	FieldMACDLine
	FieldMACDSignal
	FieldMACDHist
)

// Strategy is the unified pluggable strategy interface (Requirement 1.1).
type Strategy interface {
	Name() string                 // display name (uniqueness managed by registry)
	Params() []ParamSpec          // declared configurable parameters (Requirement 1.5)
	MinBars() int                 // minimum K线 count required (Requirement 5.5)
	Decide(ctx *Context) Decision // per-bar decision (Requirement 1.2)
}
