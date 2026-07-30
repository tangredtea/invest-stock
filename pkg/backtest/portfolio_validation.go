package backtest

import (
	"fmt"

	"invest/pkg/data"
)

func validateAlignedData(aligned AlignedData) error {
	n := len(aligned.Timeline)
	if n == 0 {
		return fmt.Errorf("统一时间轴为空")
	}
	if len(aligned.Symbols) < minPortfolioSymbols || len(aligned.Symbols) > maxPortfolioSymbols {
		return fmt.Errorf("标的数量 %d 超出 [%d, %d] 范围", len(aligned.Symbols), minPortfolioSymbols, maxPortfolioSymbols)
	}
	for i, ts := range aligned.Timeline {
		if ts.IsZero() {
			return fmt.Errorf("统一时间轴第 %d 项时间为空", i)
		}
		if i > 0 && !ts.After(aligned.Timeline[i-1]) {
			return fmt.Errorf("统一时间轴必须严格递增: 第 %d 项 %v 不晚于前一项 %v", i, ts, aligned.Timeline[i-1])
		}
	}

	seen := make(map[string]struct{}, len(aligned.Symbols))
	for i, sym := range aligned.Symbols {
		if sym == "" {
			return fmt.Errorf("第 %d 个标的为空", i)
		}
		if _, ok := seen[sym]; ok {
			return fmt.Errorf("标的重复: %s", sym)
		}
		if i > 0 && sym < aligned.Symbols[i-1] {
			return fmt.Errorf("标的列表必须按字典序排列: %s 在 %s 之后", sym, aligned.Symbols[i-1])
		}
		seen[sym] = struct{}{}

		points, ok := aligned.Points[sym]
		if !ok {
			return fmt.Errorf("标的 %s 缺少对齐价格点", sym)
		}
		if len(points) != n {
			return fmt.Errorf("标的 %s 对齐价格点数量 %d 与时间轴数量 %d 不一致", sym, len(points), n)
		}
		for j, p := range points {
			if err := validatePricePoint(sym, j, p); err != nil {
				return err
			}
		}

		raw, ok := aligned.Raw[sym]
		if !ok {
			return fmt.Errorf("标的 %s 缺少原始 K线", sym)
		}
		if err := data.ValidateDailyKLines(fmt.Sprintf("标的 %s 的原始 K线", sym), raw); err != nil {
			return err
		}
	}
	for sym := range aligned.Points {
		if _, ok := seen[sym]; !ok {
			return fmt.Errorf("对齐价格点包含未知标的: %s", sym)
		}
	}
	for sym := range aligned.Raw {
		if _, ok := seen[sym]; !ok {
			return fmt.Errorf("原始 K线包含未知标的: %s", sym)
		}
	}
	return nil
}

func validatePortfolioConfigSymbols(cfg PortfolioConfig, allowed map[string]struct{}) error {
	if err := rejectUnknownWeights(cfg.Weights, allowed); err != nil {
		return err
	}
	for sym := range cfg.Risk.PerSymbolCap {
		if _, ok := allowed[sym]; !ok {
			return fmt.Errorf("权重上限包含未知标的: %s", sym)
		}
	}
	return nil
}

func symbolSet(symbols []string) map[string]struct{} {
	out := make(map[string]struct{}, len(symbols))
	for _, sym := range symbols {
		out[sym] = struct{}{}
	}
	return out
}

func rejectUnknownWeights(weights map[string]float64, allowed map[string]struct{}) error {
	for sym := range weights {
		if _, ok := allowed[sym]; !ok {
			return fmt.Errorf("目标权重包含未知标的: %s", sym)
		}
	}
	return nil
}

func rejectUnknownSignals(signals map[string]Decision, allowed map[string]struct{}) error {
	for sym := range signals {
		if _, ok := allowed[sym]; !ok {
			return fmt.Errorf("交易信号包含未知标的: %s", sym)
		}
	}
	return nil
}

func validatePricePoint(sym string, i int, p PricePoint) error {
	if p.Valid {
		if p.Missing || p.Suspended {
			return fmt.Errorf("标的 %s 第 %d 个价格点状态冲突: valid 不能同时 missing/suspended", sym, i)
		}
		return validatePointPrices(sym, i, p.Open, p.High, p.Low, p.Close)
	}
	if p.Missing && p.Suspended {
		return fmt.Errorf("标的 %s 第 %d 个价格点状态冲突: missing 与 suspended 不能同时为 true", sym, i)
	}
	if p.Suspended {
		return validatePointPrices(sym, i, p.Open, p.High, p.Low, p.Close)
	}
	if p.Missing {
		if p.Open != 0 || p.High != 0 || p.Low != 0 || p.Close != 0 {
			return fmt.Errorf("标的 %s 第 %d 个缺失价格点不应携带价格", sym, i)
		}
		return nil
	}
	return fmt.Errorf("标的 %s 第 %d 个价格点状态非法", sym, i)
}

func validatePointPrices(sym string, i int, open, high, low, close float64) error {
	k := fmt.Sprintf("标的 %s 第 %d 个价格点", sym, i)
	if !finiteFloat(open) || !finiteFloat(high) || !finiteFloat(low) || !finiteFloat(close) {
		return fmt.Errorf("%s 价格必须为有限数", k)
	}
	if open <= 0 || high <= 0 || low <= 0 || close <= 0 {
		return fmt.Errorf("%s 价格必须为正数", k)
	}
	if high < open || high < close || low > open || low > close {
		return fmt.Errorf("%s OHLC 关系非法", k)
	}
	return nil
}
