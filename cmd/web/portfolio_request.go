package main

import (
	"strings"

	"invest/pkg/backtest"
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

func (req *portfolioReq) normalizeCodes() {
	for i, code := range req.Codes {
		req.Codes[i] = strings.TrimSpace(code)
	}
}

func (req *portfolioReq) normalizeWeightKeys() {
	req.Weights = normalizeFloatMapKeys(req.Weights)
	if req.Risk != nil {
		req.Risk.PerSymbolCap = normalizeFloatMapKeys(req.Risk.PerSymbolCap)
	}
}

func normalizeFloatMapKeys(in map[string]float64) map[string]float64 {
	if len(in) == 0 {
		return in
	}
	out := make(map[string]float64, len(in))
	for key, value := range in {
		out[strings.TrimSpace(key)] = value
	}
	return out
}

func hasDuplicateNormalizedFloatKeys(in map[string]float64) (string, bool) {
	seen := make(map[string]struct{}, len(in))
	for key := range in {
		normalized := strings.TrimSpace(key)
		if _, ok := seen[normalized]; ok {
			return normalized, true
		}
		seen[normalized] = struct{}{}
	}
	return "", false
}

func firstUnknownFloatKey(in map[string]float64, allowed map[string]struct{}) (string, bool) {
	for key := range in {
		if _, ok := allowed[key]; !ok {
			return key, true
		}
	}
	return "", false
}

func stringSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		out[value] = struct{}{}
	}
	return out
}

func duplicateCode(codes []string) (string, bool) {
	seen := make(map[string]struct{}, len(codes))
	for _, code := range codes {
		if _, ok := seen[code]; ok {
			return code, true
		}
		seen[code] = struct{}{}
	}
	return "", false
}
