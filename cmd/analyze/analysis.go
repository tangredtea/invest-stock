package main

import (
	"fmt"
	"sort"

	"invest/pkg/data"
)

type thresholdStat struct {
	Threshold float64
	Count     int
	Percent   float64
}

type analysisResult struct {
	Count       int
	Average     float64
	Median      float64
	P25         float64
	P75         float64
	Max         float64
	Thresholds  []thresholdStat
	TotalProfit float64
	DailyProfit float64
	WinRate     float64
}

func analyzeKLines(klines []data.KLine) (analysisResult, error) {
	if len(klines) <= 60 {
		return analysisResult{}, fmt.Errorf("k线数量不足: 实际 %d 根, 至少需要 61 根", len(klines))
	}

	spreads := make([]float64, 0, len(klines)-60)
	totalProfit := 0.0
	wins := 0
	for i, k := range klines[60:] {
		if k.Close <= 0 {
			return analysisResult{}, fmt.Errorf("k线收盘价无效: 第 %d 根", i+60)
		}
		spread := (k.High - k.Low) / k.Close * 100
		spreads = append(spreads, spread)

		// Simplified T+0 simulation: buy 1000 shares near daily low, sell at close.
		profit := (k.Close - k.Low) * 1000
		if profit > 0 {
			wins++
		}
		totalProfit += profit
	}
	sort.Float64s(spreads)

	sum := 0.0
	for _, s := range spreads {
		sum += s
	}

	n := len(spreads)
	result := analysisResult{
		Count:       n,
		Average:     sum / float64(n),
		Median:      spreads[n/2],
		P25:         spreads[n/4],
		P75:         spreads[n*3/4],
		Max:         spreads[n-1],
		TotalProfit: totalProfit,
		DailyProfit: totalProfit / float64(n),
		WinRate:     float64(wins) / float64(n) * 100,
	}
	for _, th := range []float64{0.5, 0.8, 1.0, 1.5} {
		cnt := 0
		for _, s := range spreads {
			if s >= th {
				cnt++
			}
		}
		result.Thresholds = append(result.Thresholds, thresholdStat{
			Threshold: th,
			Count:     cnt,
			Percent:   float64(cnt) / float64(n) * 100,
		})
	}
	return result, nil
}
