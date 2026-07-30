package analysis

import (
	"fmt"

	"invest/pkg/data"
	"invest/pkg/indicator"
	"invest/pkg/strategy"
)

type Snapshot struct {
	Closes   []float64
	Series   indicator.Series
	Current  indicator.Indicators
	Previous indicator.Indicators
}

const DefaultSignalHoldings = 100000

func (s Snapshot) LastClose() float64 {
	return s.Closes[len(s.Closes)-1]
}

func (s Snapshot) LastIndex() int {
	return len(s.Closes) - 1
}

// LatestIndicators computes the indicator series once and returns snapshots for
// the latest and previous bars.
func LatestIndicators(klines []data.KLine) (Snapshot, bool) {
	if len(klines) < 2 {
		return Snapshot{}, false
	}
	closes := make([]float64, len(klines))
	for i, k := range klines {
		closes[i] = k.Close
	}
	series := indicator.ComputeSeries(closes)
	last := len(closes) - 1
	return Snapshot{
		Closes:   closes,
		Series:   series,
		Current:  indicator.At(series, last),
		Previous: indicator.At(series, last-1),
	}, true
}

// RequireLatestIndicators returns the latest indicator snapshot or a
// user-facing error explaining why it cannot be computed.
func RequireLatestIndicators(klines []data.KLine) (Snapshot, error) {
	snapshot, ok := LatestIndicators(klines)
	if !ok {
		return Snapshot{}, fmt.Errorf("k线数量不足: 实际 %d 根, 至少需要 2 根", len(klines))
	}
	return snapshot, nil
}

func CompositeSignal(snapshot Snapshot, quote data.Quote, holdings int) strategy.SignalResult {
	qi := strategy.QuoteInfo{
		Price:    quote.Price,
		High:     quote.High,
		Low:      quote.Low,
		PreClose: quote.PreClose,
	}
	return strategy.Composite(
		snapshot.LastClose(),
		snapshot.Current.MA5,
		snapshot.Current.MA20,
		snapshot.Current.MA60,
		snapshot.Current.RSI14,
		snapshot.Current.BollUpper,
		snapshot.Current.BollLower,
		snapshot.Current.MACDHist,
		snapshot.Previous.MACDHist,
		holdings,
		qi,
	)
}

func DefaultCompositeSignal(snapshot Snapshot, quote data.Quote) strategy.SignalResult {
	return CompositeSignal(snapshot, quote, DefaultSignalHoldings)
}
