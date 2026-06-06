package main

import (
	"encoding/json"
	"net/http"

	"invest/pkg/backtest"
	_ "invest/pkg/backtest/builtin" // register builtin strategies into backtest.Default
	"invest/pkg/data"
)

// portfolioReq is a multi-symbol portfolio backtest request (Requirement 7.1).
type portfolioReq struct {
	Codes     []string                       `json:"codes"`     // 2..50 symbols
	Strategy  string                         `json:"strategy"`  // builtin name (when PerSymbol) else ignored
	PerSymbol bool                           `json:"perSymbol"` // true => wrap a builtin per symbol
	Params    map[string]backtest.ParamValue `json:"params"`
	Policy    int                            `json:"policy"`  // AlignmentPolicy
	Scheme    int                            `json:"scheme"`  // WeightScheme
	Weights   map[string]float64             `json:"weights"` // specified weights (keyed by code)
	Rebalance *rebalanceReq                  `json:"rebalance"`
	Risk      *riskReq                       `json:"risk"`
	Config    *backtestConfigReq             `json:"config"`
}

type rebalanceReq struct {
	Periodic       bool    `json:"periodic"`
	PeriodBars     int     `json:"periodBars"`
	Threshold      bool    `json:"threshold"`
	ThresholdValue float64 `json:"thresholdValue"`
}

type riskReq struct {
	StopLossEnabled     bool               `json:"stopLossEnabled"`
	StopLossPct         float64            `json:"stopLossPct"`
	StopLossByPeak      bool               `json:"stopLossByPeak"`
	TakeProfitEnabled   bool               `json:"takeProfitEnabled"`
	TakeProfitPct       float64            `json:"takeProfitPct"`
	MaxDDGuardEnabled   bool               `json:"maxDDGuardEnabled"`
	MaxDDThreshold      float64            `json:"maxDDThreshold"`
	MaxDDTargetExposure float64            `json:"maxDDTargetExposure"`
	PerSymbolCap        map[string]float64 `json:"perSymbolCap"`
	CashReserve         float64            `json:"cashReserve"`
}

// handlePortfolioBacktest runs a portfolio backtest (Requirement 7).
func handlePortfolioBacktest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, jsonResp{Code: 405, Message: "method not allowed"})
		return
	}
	var req portfolioReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, jsonResp{Code: 400, Message: "请求格式错误"})
		return
	}
	// Requirement 7.1, 7.4: symbol count in [2,50].
	if len(req.Codes) < 2 || len(req.Codes) > 50 {
		writeJSON(w, 400, jsonResp{Code: 400, Message: "组合标的数量必须在 2 到 50 之间"})
		return
	}

	// Load each symbol's klines (Requirement 7.9).
	input := make(map[string][]data.KLine, len(req.Codes))
	for _, code := range req.Codes {
		secid := resolveSecID(code)
		if secid == "" {
			writeJSON(w, 400, jsonResp{Code: 400, Message: "无效股票代码: " + code})
			return
		}
		klines, err := data.FetchKLines(secid)
		if err != nil {
			writeJSON(w, 400, jsonResp{Code: 400, Message: "无法获取标的 " + code + " 的历史数据: " + err.Error()})
			return
		}
		input[code] = klines
	}

	cfg, err := req.toPortfolioConfig()
	if err != nil {
		writeJSON(w, 400, jsonResp{Code: 400, Message: err.Error()})
		return
	}

	aligned, err := backtest.Align(input, cfg.Policy)
	if err != nil {
		writeJSON(w, 400, jsonResp{Code: 400, Message: err.Error()})
		return
	}

	strat, err := req.buildStrategy(aligned.Symbols)
	if err != nil {
		writeJSON(w, 400, jsonResp{Code: 400, Message: err.Error()})
		return
	}

	res, err := backtest.PortfolioEngine{}.Run(aligned, strat, cfg)
	if err != nil {
		writeJSON(w, 400, jsonResp{Code: 400, Message: err.Error()})
		return
	}
	writeJSON(w, 200, jsonResp{Code: 0, Message: "ok", Data: res})
}

// toPortfolioConfig builds a backtest.PortfolioConfig from the request.
func (req portfolioReq) toPortfolioConfig() (backtest.PortfolioConfig, error) {
	cfg := backtest.DefaultPortfolioConfig()
	if req.Config != nil {
		base, err := req.Config.toConfig()
		if err != nil {
			return cfg, err
		}
		cfg.Base = base
	}
	cfg.Policy = backtest.AlignmentPolicy(req.Policy)
	cfg.Scheme = backtest.WeightScheme(req.Scheme)
	cfg.Weights = req.Weights
	if req.Rebalance != nil {
		cfg.Rebalance = backtest.RebalanceConfig{
			Periodic: req.Rebalance.Periodic, PeriodBars: req.Rebalance.PeriodBars,
			Threshold: req.Rebalance.Threshold, ThresholdValue: req.Rebalance.ThresholdValue,
		}
	}
	if req.Risk != nil {
		ref := backtest.StopLossByCost
		if req.Risk.StopLossByPeak {
			ref = backtest.StopLossByPeak
		}
		cfg.Risk = backtest.RiskConfig{
			StopLossEnabled: req.Risk.StopLossEnabled, StopLossPct: req.Risk.StopLossPct, StopLossRef: ref,
			TakeProfitEnabled: req.Risk.TakeProfitEnabled, TakeProfitPct: req.Risk.TakeProfitPct,
			MaxDDGuardEnabled: req.Risk.MaxDDGuardEnabled, MaxDDThreshold: req.Risk.MaxDDThreshold,
			MaxDDTargetExposure: req.Risk.MaxDDTargetExposure,
			PerSymbolCap:        req.Risk.PerSymbolCap, CashReserve: req.Risk.CashReserve,
		}
	}
	return cfg, nil
}

// buildStrategy builds either a per-symbol adapter (wrapping a builtin) or the
// scheme-weights portfolio strategy.
func (req portfolioReq) buildStrategy(symbols []string) (backtest.PortfolioStrategy, error) {
	if req.PerSymbol {
		return backtest.NewPerSymbolAdapter(backtest.Default, req.Strategy, req.Params, symbols)
	}
	return backtest.SchemeStrategy{}, nil
}
