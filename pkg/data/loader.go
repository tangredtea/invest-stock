package data

import (
	"fmt"
	"net/url"
	"sync"
	"time"
)

func klineURL(secid string) string {
	q := url.Values{}
	q.Set("secid", secid)
	q.Set("fields1", "f1,f2,f3,f4,f5,f6")
	q.Set("fields2", "f51,f52,f53,f54,f55,f56,f57")
	q.Set("klt", "101")
	q.Set("fqt", "0")
	q.Set("beg", "20231201")
	q.Set("end", "20991231")
	return "https://push2his.eastmoney.com/api/qt/stock/kline/get?" + q.Encode()
}

type apiResp struct {
	Data struct {
		Klines []string `json:"klines"`
	} `json:"data"`
}

// defaultKLineCache caches daily K-line data. It is guarded because tests and
// startup code may replace it while request handlers are already using it.
var (
	defaultKLineCacheMu sync.RWMutex
	defaultKLineCache   = NewKLineCache(defaultKLineTTL)
)

// SetKLineCacheTTL replaces the default K-line cache with one using the given
// TTL (clamped to [1s, 24h]).
func SetKLineCacheTTL(ttl time.Duration) {
	defaultKLineCacheMu.Lock()
	defaultKLineCache = NewKLineCache(ttl)
	defaultKLineCacheMu.Unlock()
}

// SeedKLineCache preloads the default cache with K-line data for a secid.
// Useful for tests (to avoid network calls) and for warm-starting from a local
// store. The data is served until its TTL expires.
func SeedKLineCache(secid string, klines []KLine) {
	defaultKLineCacheMu.RLock()
	cache := defaultKLineCache
	defaultKLineCacheMu.RUnlock()
	cache.Set(secid, klines)
}

// FetchKLines fetches daily K-line data from EastMoney API, serving from an
// in-memory TTL cache when a fresh entry exists.
func FetchKLines(secid string) ([]KLine, error) {
	rawSecID := secid
	secid, ok := normalizeSecID(secid)
	if !ok {
		return nil, fmt.Errorf("无效证券ID: %q", rawSecID)
	}

	defaultKLineCacheMu.RLock()
	cache := defaultKLineCache
	defaultKLineCacheMu.RUnlock()

	if cached, ok := cache.Get(secid); ok {
		return cached, nil
	}

	body, err := httpGet(klineURL(secid))
	if err != nil {
		return nil, fmt.Errorf("HTTP请求失败: %w", err)
	}

	lines, err := parseKLinesResponse(body)
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return nil, fmt.Errorf("未获取到K线数据")
	}
	if err := ValidateDailyKLines("K线", lines); err != nil {
		return nil, err
	}
	cache.Set(secid, lines)
	return lines, nil
}
