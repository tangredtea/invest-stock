package strategy

// Trend represents market trend direction.
type Trend int

const (
	TrendUp   Trend = 1
	TrendDown Trend = -1
	TrendFlat Trend = 0
)

// DetectTrend determines trend from MA crossovers.
// ma5/ma20 for short-term, ma20/ma60 for mid-term.
func DetectTrend(ma5, ma20, ma60 float64) Trend {
	if ma60 == 0 {
		return TrendFlat
	}
	shortUp := ma5 > ma20
	midUp := ma20 > ma60

	if shortUp && midUp {
		return TrendUp
	}
	if !shortUp && !midUp {
		return TrendDown
	}
	return TrendFlat
}

func (t Trend) String() string {
	switch t {
	case TrendUp:
		return "上升趋势"
	case TrendDown:
		return "下降趋势"
	default:
		return "震荡"
	}
}
