package backtest

import (
	"fmt"

	"invest/pkg/data"
	"invest/pkg/indicator"
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
	case Buy:
		return "buy"
	case Sell:
		return "sell"
	default:
		return "hold"
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
func BuyQty(qty int) Decision        { return Decision{Action: Buy, Qty: qty} }
func BuyAmount(amount float64) Decision { return Decision{Action: Buy, Amount: amount} }
func SellQty(qty int) Decision       { return Decision{Action: Sell, Qty: qty} }
func SellAmount(amount float64) Decision { return Decision{Action: Sell, Amount: amount} }

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

// Context is the read-only context the engine provides to a strategy on bar Index.
// Strategies may only read market data at indices <= Index; reading the future
// panics (Requirement 1.9), which surfaces look-ahead bugs in tests.
//
// Cash and Shares expose a read-only snapshot of the account state *before* the
// current bar's order is matched, so position-aware strategies (full-position
// entry/exit, grid sizing) can size their orders.
type Context struct {
	Index  int
	Klines []data.KLine
	Series indicator.Series
	Cash   float64 // available cash before this bar's fill
	Shares int     // total holdings before this bar's fill
}

// Bar returns the i-th K线; panics if i > Index (look-ahead violation).
func (c *Context) Bar(i int) data.KLine {
	if i < 0 || i > c.Index {
		panic(fmt.Sprintf("backtest: look-ahead access Bar(%d) at index %d", i, c.Index))
	}
	return c.Klines[i]
}

// Ind returns indicator field value at index i; panics if i > Index.
func (c *Context) Ind(field IndField, i int) float64 {
	if i < 0 || i > c.Index {
		panic(fmt.Sprintf("backtest: look-ahead access Ind(%d,%d) at index %d", field, i, c.Index))
	}
	s := c.Series
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

// Strategy is the unified pluggable strategy interface (Requirement 1.1).
type Strategy interface {
	Name() string                 // display name (uniqueness managed by registry)
	Params() []ParamSpec          // declared configurable parameters (Requirement 1.5)
	MinBars() int                 // minimum K线 count required (Requirement 5.5)
	Decide(ctx *Context) Decision // per-bar decision (Requirement 1.2)
}

// ParamType is the set of predefined parameter types (Requirement 1.5).
type ParamType int

const (
	ParamInt ParamType = iota
	ParamFloat
	ParamEnum
	ParamBool
)

func (t ParamType) String() string {
	switch t {
	case ParamInt:
		return "int"
	case ParamFloat:
		return "float"
	case ParamEnum:
		return "enum"
	case ParamBool:
		return "bool"
	default:
		return "unknown"
	}
}

// ParamValue is a tagged parameter value (avoids interface{} coercion issues).
type ParamValue struct {
	Type  ParamType `json:"type"`
	Int   int       `json:"int,omitempty"`
	Float float64   `json:"float,omitempty"`
	Str   string    `json:"str,omitempty"`
	Bool  bool      `json:"bool,omitempty"`
}

func IntVal(v int) ParamValue     { return ParamValue{Type: ParamInt, Int: v} }
func FloatVal(v float64) ParamValue { return ParamValue{Type: ParamFloat, Float: v} }
func EnumVal(v string) ParamValue { return ParamValue{Type: ParamEnum, Str: v} }
func BoolVal(v bool) ParamValue   { return ParamValue{Type: ParamBool, Bool: v} }

// ParamSpec declares a single parameter (Requirement 1.5).
type ParamSpec struct {
	Name    string       `json:"name"`    // unique and non-empty
	Type    ParamType    `json:"type"`
	Default ParamValue   `json:"default"` // must lie within range
	Min     float64      `json:"min"`     // inclusive lower bound for Int/Float
	Max     float64      `json:"max"`     // inclusive upper bound for Int/Float
	Enum    []string     `json:"enum,omitempty"` // allowed values for Enum
	Desc    string       `json:"desc,omitempty"`
}

// ApplyDefaults merges user-provided params with declared defaults and validates:
//   - missing params get their default (Requirement 1.6)
//   - out-of-range -> error naming the param (Requirement 1.7)
//   - unknown name or type mismatch -> error (Requirement 1.8)
// Returns a validated complete param map; on any error returns nil + error and
// never constructs an instance.
func ApplyDefaults(specs []ParamSpec, in map[string]ParamValue) (map[string]ParamValue, error) {
	known := make(map[string]ParamSpec, len(specs))
	for _, s := range specs {
		known[s.Name] = s
	}
	// Reject unknown parameter names (Requirement 1.8).
	for name := range in {
		if _, ok := known[name]; !ok {
			return nil, fmt.Errorf("未知参数: %q", name)
		}
	}

	out := make(map[string]ParamValue, len(specs))
	for _, s := range specs {
		v, provided := in[s.Name]
		if !provided {
			out[s.Name] = s.Default // Requirement 1.6
			continue
		}
		if v.Type != s.Type {
			return nil, fmt.Errorf("参数 %q 类型不匹配: 期望 %s, 实际 %s", s.Name, s.Type, v.Type)
		}
		if err := validateValue(s, v); err != nil {
			return nil, err
		}
		out[s.Name] = v
	}
	return out, nil
}

// validateValue checks a provided value against its spec's range/enum.
func validateValue(s ParamSpec, v ParamValue) error {
	switch s.Type {
	case ParamInt:
		f := float64(v.Int)
		if f < s.Min || f > s.Max {
			return fmt.Errorf("参数 %q 越界: %d 不在 [%g, %g]", s.Name, v.Int, s.Min, s.Max)
		}
	case ParamFloat:
		if v.Float < s.Min || v.Float > s.Max {
			return fmt.Errorf("参数 %q 越界: %g 不在 [%g, %g]", s.Name, v.Float, s.Min, s.Max)
		}
	case ParamEnum:
		ok := false
		for _, e := range s.Enum {
			if e == v.Str {
				ok = true
				break
			}
		}
		if !ok {
			return fmt.Errorf("参数 %q 取值非法: %q 不在枚举集合内", s.Name, v.Str)
		}
	case ParamBool:
		// any bool is valid
	}
	return nil
}
