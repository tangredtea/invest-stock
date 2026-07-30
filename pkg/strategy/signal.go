package strategy

type Signal int

const (
	StrongBuy  Signal = 2
	Buy        Signal = 1
	Hold       Signal = 0
	Sell       Signal = -1
	StrongSell Signal = -2
)

func (s Signal) String() string {
	switch s {
	case StrongBuy:
		return "强买入"
	case Buy:
		return "买入"
	case Hold:
		return "持有"
	case Sell:
		return "卖出"
	case StrongSell:
		return "强卖出"
	default:
		return "未知"
	}
}

// T0Direction indicates intraday T+0 operation direction.
type T0Direction int

const (
	T0BuyFirst  T0Direction = 1  // 先买后卖(看涨日内)
	T0SellFirst T0Direction = -1 // 先卖后买(看跌日内)
	T0Skip      T0Direction = 0  // 不做T+0
)

func (d T0Direction) String() string {
	switch d {
	case T0BuyFirst:
		return "正T(先买后卖)"
	case T0SellFirst:
		return "反T(先卖后买)"
	default:
		return "观望不做"
	}
}

type SignalResult struct {
	Trend       Trend       `json:"trend"`
	DCASignal   Signal      `json:"dcaSignal"`
	T0Dir       T0Direction `json:"t0Dir"`
	T0Shares    int         `json:"t0Shares"`
	T0BuyPrice  float64     `json:"t0BuyPrice"`
	T0SellPrice float64     `json:"t0SellPrice"`
	T0Reasons   []string    `json:"t0Reasons"`
	Reason      string      `json:"reason"`
}

// QuoteInfo holds real-time intraday data for T+0 price calculation.
type QuoteInfo struct {
	Price, High, Low, PreClose float64
}
