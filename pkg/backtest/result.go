// Package backtest provides a deterministic, event-driven backtesting engine,
// a pluggable strategy framework, and a full suite of performance metrics.
//
// The engine is a pure function of its inputs (klines + strategy + params +
// config): it never reads the real clock, random numbers, or the network, so
// identical inputs always yield bit-for-bit identical results.
package backtest

import "time"

// TradeRecord is a single executed fill (Requirements 7.9, 14.2, 17.5).
type TradeRecord struct {
	Date       time.Time `json:"date"`       // K线日期 of the bar the fill occurred on
	Action     string    `json:"action"`     // "buy" / "sell"
	Price      float64   `json:"price"`      // executed price (slippage included)
	BasePrice  float64   `json:"basePrice"`  // Fill_Rule base price before slippage
	Qty        int       `json:"qty"`        // executed quantity (shares)
	Turnover   float64   `json:"turnover"`   // Price * Qty
	Commission float64   `json:"commission"` // brokerage fee
	StampTax   float64   `json:"stampTax"`   // stamp tax (sell side only)
	Slippage   float64   `json:"slippage"`   // |Price-BasePrice| * Qty
	CostTotal  float64   `json:"costTotal"`  // Commission + StampTax + Slippage
	RealizedPL float64   `json:"realizedPL"` // realized P/L for a closing (sell), net of cost
}

// UnfilledOrder records an order that was abandoned (Requirements 5.4, 7.8).
type UnfilledOrder struct {
	Date   time.Time `json:"date"`
	Action string    `json:"action"`
	Reason string    `json:"reason"` // "no_next_bar" / "insufficient_cash" / "tplus1_rejected"
	Qty    int       `json:"qty"`
}

// Metrics aggregates all performance metrics (Requirements 9–12).
type Metrics struct {
	TotalReturn      float64   `json:"totalReturn"`      // (final-initial)/initial
	AnnualReturn     float64   `json:"annualReturn"`     // (1+total)^(365/days)-1
	MaxDrawdown      float64   `json:"maxDrawdown"`      // in [0,1]
	MaxDDPeakDate    time.Time `json:"maxDDPeakDate"`    // peak date of max drawdown
	MaxDDTroughDate  time.Time `json:"maxDDTroughDate"`  // trough date of max drawdown
	LongestDDBars    int       `json:"longestDDBars"`    // longest drawdown duration (bars)
	AnnualVolatility float64   `json:"annualVolatility"` // sample std * sqrt(252)
	Sharpe           float64   `json:"sharpe"`           // annualized excess / annualized vol
	Sortino          float64   `json:"sortino"`          // annualized excess / downside vol
	Calmar           float64   `json:"calmar"`           // annual return / |max drawdown|
	TotalTrades      int       `json:"totalTrades"`      // number of TradeRecords
	WinRate          float64   `json:"winRate"`          // in [0,1]
	ProfitLossRatio  float64   `json:"profitLossRatio"`  // avgWin / |avgLoss|
	AvgWin           float64   `json:"avgWin"`           // average winning realized P/L
	AvgLoss          float64   `json:"avgLoss"`          // average losing realized P/L
	Turnover         float64   `json:"turnover"`         // sum(turnover) / initial cash
}

// Result is the complete output of a backtest (Requirements 5.6, 14.2).
type Result struct {
	StrategyName string                `json:"strategyName"`
	Params       map[string]ParamValue `json:"params"`   // parameters used in this run
	Equity       []float64             `json:"equity"`   // first value = initial cash, len = #bars
	Drawdown     []float64             `json:"drawdown"` // first value = 0, each in [0,1]
	NetValue     []float64             `json:"netValue"` // first value = 1
	Dates        []time.Time           `json:"dates"`    // aligned with the curves
	Trades       []TradeRecord         `json:"trades"`
	Unfilled     []UnfilledOrder       `json:"unfilled"`
	Metrics      Metrics               `json:"metrics"`
}
