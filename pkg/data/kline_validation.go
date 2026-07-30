package data

import (
	"fmt"
	"math"
)

// ValidateKLines checks a time series for finite OHLCV values and strictly
// ascending timestamps. Use ValidateDailyKLines when the series is keyed by
// trading day and duplicate same-day bars must also be rejected.
func ValidateKLines(label string, klines []KLine) error {
	if label == "" {
		label = "K线"
	}
	for i, k := range klines {
		if k.Date.IsZero() {
			return fmt.Errorf("%s时间序列无效: 第 %d 根日期为空", label, i)
		}
		if err := validateKLineValues(label, i, k); err != nil {
			return err
		}
		if i > 0 && !k.Date.After(klines[i-1].Date) {
			return fmt.Errorf("%s时间序列无效: 第 %d 根日期非严格升序或重复", label, i)
		}
	}
	return nil
}

// ValidateDailyKLines additionally rejects multiple bars on the same trading
// day, matching the daily alignment key used by portfolio backtests.
func ValidateDailyKLines(label string, klines []KLine) error {
	if err := ValidateKLines(label, klines); err != nil {
		return err
	}
	for i := 1; i < len(klines); i++ {
		if !DayKey(klines[i].Date).After(DayKey(klines[i-1].Date)) {
			return fmt.Errorf("%s时间序列无效: 第 %d 根交易日非严格升序或重复", label, i)
		}
	}
	return nil
}

func validateKLineValues(label string, i int, k KLine) error {
	if !finiteFloat(k.Open) || !finiteFloat(k.High) || !finiteFloat(k.Low) || !finiteFloat(k.Close) || !finiteFloat(k.Volume) {
		return fmt.Errorf("%s数值无效: 第 %d 根包含 NaN 或 Inf", label, i)
	}
	if k.Open <= 0 || k.High <= 0 || k.Low <= 0 || k.Close <= 0 {
		return fmt.Errorf("%s价格无效: 第 %d 根 OHLC 必须全部为正数", label, i)
	}
	if k.Volume < 0 {
		return fmt.Errorf("%s成交量无效: 第 %d 根成交量不能为负数", label, i)
	}
	if k.High < k.Low {
		return fmt.Errorf("%s价格关系无效: 第 %d 根最高价低于最低价", label, i)
	}
	if k.High < k.Open || k.High < k.Close {
		return fmt.Errorf("%s价格关系无效: 第 %d 根最高价低于开盘价或收盘价", label, i)
	}
	if k.Low > k.Open || k.Low > k.Close {
		return fmt.Errorf("%s价格关系无效: 第 %d 根最低价高于开盘价或收盘价", label, i)
	}
	return nil
}

func finiteFloat(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}
