package main

import (
	"net/http"
	"strconv"

	"invest/pkg/backtest"
)

// handleBacktest runs a single-strategy backtest (Requirement 14).
func handleBacktest(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var req backtestReq
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, 400, jsonResp{Code: 400, Message: "请求格式错误"})
		return
	}

	klines, status, errMsg := loadKlinesForBacktest(req.Code)
	if errMsg != "" {
		writeJSON(w, status, jsonResp{Code: status, Message: errMsg})
		return
	}

	res, status, errMsg := runOneBacktest(req, klines)
	if errMsg != "" {
		writeJSON(w, status, jsonResp{Code: status, Message: errMsg})
		return
	}
	writeJSON(w, 200, jsonResp{Code: 0, Message: "ok", Data: res})
}

// handleBacktestCompare runs multiple strategies on the same data/config and
// returns results in request order (Requirement 15).
func handleBacktestCompare(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var req backtestCompareReq
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, 400, jsonResp{Code: 400, Message: "请求格式错误"})
		return
	}
	if len(req.Strategies) < 1 || len(req.Strategies) > 50 {
		writeJSON(w, 400, jsonResp{Code: 400, Message: "策略组合数量必须在 1 到 50 之间"})
		return
	}

	klines, status, errMsg := loadKlinesForBacktest(req.Code)
	if errMsg != "" {
		writeJSON(w, status, jsonResp{Code: status, Message: errMsg})
		return
	}

	results := make([]backtest.Result, 0, len(req.Strategies))
	for i, item := range req.Strategies {
		single := backtestReq{Code: req.Code, Strategy: item.Strategy, Params: item.Params, Config: req.Config}
		res, st, em := runOneBacktest(single, klines)
		if em != "" {
			writeJSON(w, st, jsonResp{Code: st, Message: "组合 #" + strconv.Itoa(i) + " (" + item.Strategy + "): " + em})
			return
		}
		results = append(results, res)
	}
	writeJSON(w, 200, jsonResp{Code: 0, Message: "ok", Data: results})
}

// handleListStrategies returns all registered strategies with their parameter
// definitions, so the frontend can render parameter forms (Requirement 18.2).
func handleListStrategies(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	writeJSON(w, 200, jsonResp{Code: 0, Message: "ok", Data: backtest.Default.List()})
}
