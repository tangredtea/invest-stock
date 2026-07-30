package main

import (
	"invest/internal/analysis"
	"invest/pkg/data"
	"invest/pkg/indicator"
	"invest/pkg/strategy"
)

type analyzeResp struct {
	Code         string                    `json:"code"`
	SecID        string                    `json:"secid"`
	Klines       []data.KLine              `json:"klines"`
	Indicators   indicator.Series          `json:"indicators"`
	Current      indicator.Indicators      `json:"current"`
	Signal       strategy.SignalResult     `json:"signal"`
	Backtest     []strategy.BacktestResult `json:"backtest"`
	QuoteWarning string                    `json:"quoteWarning,omitempty"`
}

type analyzeData struct {
	Code     string
	SecID    string
	Klines   []data.KLine
	Snapshot analysis.Snapshot
}

type analyzeClientError int

const (
	analyzeClientErrorNone analyzeClientError = iota
	analyzeClientErrorInvalidCode
	analyzeClientErrorInsufficientKLines
)

func loadAnalyzeData(code string) (analyzeData, analyzeClientError, error) {
	loadedKLines, klineErr, err := loadKLinesForCode(code)
	switch klineErr {
	case klineClientErrorInvalidCode:
		return analyzeData{}, analyzeClientErrorInvalidCode, nil
	case klineClientErrorFetchFailed:
		return analyzeData{}, analyzeClientErrorNone, err
	}
	snapshot, ok := analysis.LatestIndicators(loadedKLines.KLines)
	if !ok {
		return analyzeData{}, analyzeClientErrorInsufficientKLines, nil
	}
	return analyzeData{
		Code:     code,
		SecID:    loadedKLines.SecID,
		Klines:   loadedKLines.KLines,
		Snapshot: snapshot,
	}, analyzeClientErrorNone, nil
}

func analyzeHTTPErrorMessage(err analyzeClientError) string {
	switch err {
	case analyzeClientErrorInvalidCode:
		return "无效股票代码"
	case analyzeClientErrorInsufficientKLines:
		return "K线数量不足,无法分析"
	default:
		return ""
	}
}

func analyzeWSErrorMessage(err analyzeClientError) string {
	switch err {
	case analyzeClientErrorInvalidCode:
		return "无效股票代码"
	case analyzeClientErrorInsufficientKLines:
		return "K线数量不足,无法监控"
	default:
		return ""
	}
}
