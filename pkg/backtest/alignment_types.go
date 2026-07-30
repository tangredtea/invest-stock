package backtest

import (
	"time"

	"invest/pkg/data"
)

// AlignmentPolicy selects how multiple symbols' K线 series are aligned onto a
// unified timeline (Requirements 1.2, 1.3, 1.10).
type AlignmentPolicy int

const (
	AlignIntersection AlignmentPolicy = iota // intersection of trading days
	AlignUnionFFill                          // union + forward-fill
)

const (
	minPortfolioSymbols = 2
	maxPortfolioSymbols = 1000
)

// PricePoint is a symbol's aligned price state at one timeline point
// (Requirements 1.2, 1.3, 1.4).
type PricePoint struct {
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Close     float64 `json:"close"` // forward-filled to last valid close when Suspended
	Valid     bool    `json:"valid"` // true => tradable at this point
	Suspended bool    `json:"suspended"`
	Missing   bool    `json:"missing"`
}

// AlignedData is the result of aligning multiple symbols (Requirement 1).
type AlignedData struct {
	Symbols  []string                // stable sorted order (determinism)
	Timeline []time.Time             // strictly ascending, no duplicates
	Points   map[string][]PricePoint // each symbol: len == len(Timeline)
	Raw      map[string][]data.KLine // validated raw input (for indicator windows)
}
