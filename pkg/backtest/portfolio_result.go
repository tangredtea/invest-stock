package backtest

import "time"

// PortfolioTrade is a single fill within a portfolio backtest, reusing the
// single-symbol TradeRecord and tagging it with the symbol and whether it was
// produced by a rebalance (Requirements 3.7, 4.x).
type PortfolioTrade struct {
	Symbol    string `json:"symbol"`
	TradeRecord
	Rebalance bool `json:"rebalance"`
}

// PortfolioUnfilled records an abandoned order within a portfolio backtest.
type PortfolioUnfilled struct {
	Symbol string `json:"symbol"`
	UnfilledOrder
}

// Attribution is a single symbol's contribution to total portfolio return and
// its final weight (Requirements 6.3, 6.4).
type Attribution struct {
	Symbol       string  `json:"symbol"`
	Contribution float64 `json:"contribution"` // (realized + unrealized P/L) / initial cash
	FinalWeight  float64 `json:"finalWeight"`  // end-of-backtest weight
}

// PortfolioResult is the complete output of a portfolio backtest
// (Requirements 3.7, 6, 7.3).
type PortfolioResult struct {
	Symbols        []string             `json:"symbols"`     // stable, sorted order
	Dates          []time.Time          `json:"dates"`       // = Timeline, length N
	Equity         []float64            `json:"equity"`      // len N, first = initial cash
	Drawdown       []float64            `json:"drawdown"`    // len N, each in [0,1]
	NetValue       []float64            `json:"netValue"`    // len N, first = 1
	Weights        map[string][]float64 `json:"weights"`     // per symbol, len N
	CashWeight     []float64            `json:"cashWeight"`  // cash ratio series, len N
	Trades         []PortfolioTrade     `json:"trades"`
	Unfilled       []PortfolioUnfilled  `json:"unfilled"`
	Metrics        Metrics              `json:"metrics"`     // reuses single-symbol Metrics
	Attribution    []Attribution        `json:"attribution"`
	RebalanceCount int                  `json:"rebalanceCount"`
	RebalanceCost  float64              `json:"rebalanceCost"` // >= 0
	Correlation    [][]float64          `json:"correlation,omitempty"` // P2
}
