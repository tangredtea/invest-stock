package data

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

func parseKLinesResponse(body []byte) ([]KLine, error) {
	var r apiResp
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("解析JSON失败: %w", err)
	}
	lines := make([]KLine, 0, len(r.Data.Klines))
	for i, s := range r.Data.Klines {
		k, ok := parseKLine(s)
		if !ok {
			return nil, fmt.Errorf("k线数据格式无效: 第 %d 条", i)
		}
		lines = append(lines, k)
	}
	return lines, nil
}

func parseKLine(raw string) (KLine, bool) {
	// API field order: date, open, close, high, low, volume, amount.
	parts := strings.Split(raw, ",")
	if len(parts) < 6 {
		return KLine{}, false
	}
	date, err := time.Parse("2006-01-02", parts[0])
	if err != nil {
		return KLine{}, false
	}
	open, err := parseFiniteFloat(parts[1])
	if err != nil {
		return KLine{}, false
	}
	closePrice, err := parseFiniteFloat(parts[2])
	if err != nil {
		return KLine{}, false
	}
	high, err := parseFiniteFloat(parts[3])
	if err != nil {
		return KLine{}, false
	}
	low, err := parseFiniteFloat(parts[4])
	if err != nil {
		return KLine{}, false
	}
	vol, err := parseFiniteFloat(parts[5])
	if err != nil {
		return KLine{}, false
	}
	if open <= 0 || closePrice <= 0 || high <= 0 || low <= 0 || vol < 0 {
		return KLine{}, false
	}
	if high < low || high < open || high < closePrice || low > open || low > closePrice {
		return KLine{}, false
	}
	return KLine{Date: date, Open: open, High: high, Low: low, Close: closePrice, Volume: vol}, true
}

func parseFiniteFloat(raw string) (float64, error) {
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, err
	}
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, fmt.Errorf("non-finite float: %s", raw)
	}
	return v, nil
}
