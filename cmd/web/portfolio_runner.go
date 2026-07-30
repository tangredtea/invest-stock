package main

import (
	"invest/pkg/backtest"
	"invest/pkg/data"
)

func runPortfolioBacktest(req *portfolioReq) (backtest.PortfolioResult, int, string) {
	if status, msg := validatePortfolioRequest(req); msg != "" {
		return backtest.PortfolioResult{}, status, msg
	}

	input, status, errMsg := loadPortfolioKLines(req.Codes)
	if errMsg != "" {
		return backtest.PortfolioResult{}, status, errMsg
	}

	cfg, err := req.toPortfolioConfig()
	if err != nil {
		return backtest.PortfolioResult{}, 400, err.Error()
	}

	aligned, err := backtest.Align(input, cfg.Policy)
	if err != nil {
		return backtest.PortfolioResult{}, 400, err.Error()
	}

	strat, err := req.buildStrategy(aligned.Symbols)
	if err != nil {
		return backtest.PortfolioResult{}, 400, err.Error()
	}

	res, err := backtest.PortfolioEngine{}.Run(aligned, strat, cfg)
	if err != nil {
		return backtest.PortfolioResult{}, 400, err.Error()
	}
	return res, 0, ""
}

func validatePortfolioRequest(req *portfolioReq) (int, string) {
	// Requirement 7.1, 7.4: symbol count in [2,50].
	if len(req.Codes) < 2 || len(req.Codes) > 50 {
		return 400, "组合标的数量必须在 2 到 50 之间"
	}
	req.normalizeCodes()
	if duplicated, ok := duplicateCode(req.Codes); ok {
		return 400, "组合标的重复: " + duplicated
	}
	if duplicated, ok := hasDuplicateNormalizedFloatKeys(req.Weights); ok {
		return 400, "指定权重标的重复: " + duplicated
	}
	if req.Risk != nil {
		if duplicated, ok := hasDuplicateNormalizedFloatKeys(req.Risk.PerSymbolCap); ok {
			return 400, "权重上限标的重复: " + duplicated
		}
	}
	req.normalizeWeightKeys()
	codeSet := stringSet(req.Codes)
	if unknown, ok := firstUnknownFloatKey(req.Weights, codeSet); ok {
		return 400, "指定权重标的不在组合中: " + unknown
	}
	if req.Risk != nil {
		if unknown, ok := firstUnknownFloatKey(req.Risk.PerSymbolCap, codeSet); ok {
			return 400, "权重上限标的不在组合中: " + unknown
		}
	}
	return 0, ""
}

func loadPortfolioKLines(codes []string) (map[string][]data.KLine, int, string) {
	input := make(map[string][]data.KLine, len(codes))
	for _, code := range codes {
		loaded, clientErr, err := loadKLinesForCode(code)
		switch clientErr {
		case klineClientErrorInvalidCode:
			return nil, 400, "无效股票代码: " + code
		case klineClientErrorFetchFailed:
			return nil, 400, "无法获取标的 " + code + " 的历史数据: " + err.Error()
		}
		input[code] = loaded.KLines
	}
	return input, 0, ""
}
