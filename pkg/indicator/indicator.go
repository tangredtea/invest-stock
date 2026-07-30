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
	if i < 0 || i >= len(closes) {
		return Indicators{}
	}
	return At(ComputeSeries(closes), i)
}

type Series struct {
	MA5        []float64 `json:"ma5"`
	MA20       []float64 `json:"ma20"`
	MA60       []float64 `json:"ma60"`
	RSI14      []float64 `json:"rsi14"`
	BollUpper  []float64 `json:"bollUpper"`
	BollMid    []float64 `json:"bollMid"`
	BollLower  []float64 `json:"bollLower"`
	MACDLine   []float64 `json:"macdLine"`
	MACDSignal []float64 `json:"macdSignal"`
	MACDHist   []float64 `json:"macdHist"`
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

// At extracts a single-point indicator snapshot from a precomputed Series.
func At(s Series, i int) Indicators {
	if i < 0 || i >= len(s.MA5) {
		return Indicators{}
	}
	return Indicators{
		MA5: s.MA5[i], MA20: s.MA20[i], MA60: s.MA60[i],
		RSI14:      s.RSI14[i],
		BollUpper:  s.BollUpper[i],
		BollMiddle: s.BollMid[i],
		BollLower:  s.BollLower[i],
		MACDLine:   s.MACDLine[i],
		MACDSignal: s.MACDSignal[i],
		MACDHist:   s.MACDHist[i],
	}
}
