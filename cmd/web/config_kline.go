package main

import (
	"fmt"
	"time"
)

// parseKLineTTL parses INVEST_KLINE_TTL, defaulting to 24h and clamping to
// [1s, 24h]. Invalid values fall back to the default.
func parseKLineTTL(raw string) time.Duration {
	if raw == "" {
		return defaultKLineTTL
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		fmt.Printf("提示: INVEST_KLINE_TTL 无法解析(%q),使用默认 24h\n", raw)
		return defaultKLineTTL
	}
	if d < minConfigKLineTTL {
		return minConfigKLineTTL
	}
	if d > maxConfigKLineTTL {
		return maxConfigKLineTTL
	}
	return d
}
