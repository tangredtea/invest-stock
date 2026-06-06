package indicator

type Indicators struct {
	MA5        float64 `json:"ma5"`
	MA20       float64 `json:"ma20"`
	MA60       float64 `json:"ma60"`
	RSI14      float64 `json:"rsi14"`
	BollUpper  float64 `json:"bollUpper"`
	BollMiddle float64 `json:"bollMiddle"`
	BollLower  float64 `json:"bollLower"`
	MACDLine   float64 `json:"macdLine"`
	MACDSignal float64 `json:"macdSignal"`
	MACDHist   float64 `json:"macdHist"`
}

func ComputeAll(closes []float64, i int) Indicators {
	ma5 := SMA(closes, 5)
	ma20 := SMA(closes, 20)
	ma60 := SMA(closes, 60)
	rsi := RSI(closes, 14)
	upper, mid, lower := BollingerBands(closes, 20, 2.0)
	ml, sl, hist := MACD(closes, 12, 26, 9)

	return Indicators{
		MA5: ma5[i], MA20: ma20[i], MA60: ma60[i],
		RSI14:      rsi[i],
		BollUpper:  upper[i],
		BollMiddle: mid[i],
		BollLower:  lower[i],
		MACDLine:   ml[i],
		MACDSignal: sl[i],
		MACDHist:   hist[i],
	}
}

type Series struct {
	MA5       []float64 `json:"ma5"`
	MA20      []float64 `json:"ma20"`
	MA60      []float64 `json:"ma60"`
	RSI14     []float64 `json:"rsi14"`
	BollUpper []float64 `json:"bollUpper"`
	BollMid   []float64 `json:"bollMid"`
	BollLower []float64 `json:"bollLower"`
	MACDLine  []float64 `json:"macdLine"`
	MACDSignal []float64 `json:"macdSignal"`
	MACDHist  []float64 `json:"macdHist"`
}

func ComputeSeries(closes []float64) Series {
	upper, mid, lower := BollingerBands(closes, 20, 2.0)
	ml, sl, hist := MACD(closes, 12, 26, 9)
	return Series{
		MA5: SMA(closes, 5), MA20: SMA(closes, 20), MA60: SMA(closes, 60),
		RSI14:     RSI(closes, 14),
		BollUpper: upper, BollMid: mid, BollLower: lower,
		MACDLine: ml, MACDSignal: sl, MACDHist: hist,
	}
}
