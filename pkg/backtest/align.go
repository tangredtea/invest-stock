package backtest

import (
	"fmt"
	"sort"
	"time"

	"invest/pkg/data"
)

// dayKey truncates a timestamp to its calendar day (alignment key).
func dayKey(t time.Time) time.Time {
	return data.DayKey(t)
}

// Align validates and aligns multiple symbols' K线 series onto a unified
// timeline (Requirements 1.1, 1.6–1.11). On any error it returns no AlignedData.
// It is a pure function: no clock, randomness, or network.
func Align(input map[string][]data.KLine, policy AlignmentPolicy) (AlignedData, error) {
	// Requirement 1.6: symbol count in [2, 1000].
	if len(input) < minPortfolioSymbols || len(input) > maxPortfolioSymbols {
		return AlignedData{}, fmt.Errorf("标的数量 %d 超出 [%d, %d] 范围", len(input), minPortfolioSymbols, maxPortfolioSymbols)
	}
	// Requirement 1.10: policy must be supported.
	if policy != AlignIntersection && policy != AlignUnionFFill {
		return AlignedData{}, fmt.Errorf("不支持的对齐策略: %d (仅支持 交集/并集+前值填充)", policy)
	}

	// Stable, sorted symbol order — the key to determinism (Requirement 3.9).
	symbols := make([]string, 0, len(input))
	for sym := range input {
		symbols = append(symbols, sym)
	}
	sort.Strings(symbols)

	// Requirement 1.8, 1.9: each symbol's daily bars are valid, strictly
	// ascending, and unique on the day key used by alignment.
	for _, sym := range symbols {
		if err := data.ValidateDailyKLines(fmt.Sprintf("标的 %s 的 K线", sym), input[sym]); err != nil {
			return AlignedData{}, err
		}
	}

	// Build the timeline per policy.
	timeline := buildTimeline(input, symbols, policy)
	if len(timeline) == 0 {
		return AlignedData{}, fmt.Errorf("所选对齐策略下无共同可用交易日")
	}

	// Build aligned price points per symbol.
	points := make(map[string][]PricePoint, len(symbols))
	for _, sym := range symbols {
		points[sym] = buildPoints(input[sym], timeline)
	}

	return AlignedData{
		Symbols:  symbols,
		Timeline: timeline,
		Points:   points,
		Raw:      input,
	}, nil
}

// buildTimeline computes the aligned timeline per policy (Requirements 1.2, 1.3).
func buildTimeline(input map[string][]data.KLine, symbols []string, policy AlignmentPolicy) []time.Time {
	if policy == AlignIntersection {
		// Count day occurrences across symbols; keep days present in all.
		counts := make(map[time.Time]int)
		for _, sym := range symbols {
			seen := make(map[time.Time]bool)
			for _, k := range input[sym] {
				d := dayKey(k.Date)
				if !seen[d] {
					seen[d] = true
					counts[d]++
				}
			}
		}
		var out []time.Time
		for d, c := range counts {
			if c == len(symbols) {
				out = append(out, d)
			}
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Before(out[j]) })
		return out
	}
	// Union + forward-fill: all distinct days, ascending.
	set := make(map[time.Time]bool)
	for _, sym := range symbols {
		for _, k := range input[sym] {
			set[dayKey(k.Date)] = true
		}
	}
	out := make([]time.Time, 0, len(set))
	for d := range set {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Before(out[j]) })
	return out
}

// buildPoints maps a symbol's K线 onto the timeline, forward-filling gaps
// (Requirements 1.3, 1.4, 1.11).
func buildPoints(ks []data.KLine, timeline []time.Time) []PricePoint {
	// Index raw klines by day.
	byDay := make(map[time.Time]data.KLine, len(ks))
	for _, k := range ks {
		byDay[dayKey(k.Date)] = k
	}
	out := make([]PricePoint, len(timeline))
	var lastValid *data.KLine
	for i, d := range timeline {
		if k, ok := byDay[d]; ok {
			kk := k
			lastValid = &kk
			out[i] = PricePoint{Open: k.Open, High: k.High, Low: k.Low, Close: k.Close, Valid: true}
			continue
		}
		if lastValid != nil {
			// Forward-fill close; mark suspended (Requirement 1.3).
			out[i] = PricePoint{
				Open: lastValid.Close, High: lastValid.Close, Low: lastValid.Close,
				Close: lastValid.Close, Valid: false, Suspended: true,
			}
		} else {
			// No prior valid K线: missing (Requirement 1.4).
			out[i] = PricePoint{Valid: false, Missing: true}
		}
	}
	return out
}
