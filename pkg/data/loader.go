package data

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// httpClient is a shared client with a finite timeout covering connection,
// request, and response-body read. Prevents a hung upstream from blocking
// goroutines indefinitely.
var httpClient = &http.Client{Timeout: 10 * time.Second}

// isRetryable reports whether err is a transient error worth retrying
// (network connection errors, EOF while reading body, timeouts). Non-transient
// errors (e.g. a permanent failure or a JSON parse error handled by callers)
// are not retried here.
func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, os.ErrDeadlineExceeded) {
		return true
	}
	// Timeouts (including those wrapped in *url.Error) are retryable.
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	// Connection-level network errors are retryable.
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}
	return false
}

// httpGet issues a GET through httpClient, retrying transient errors up to a
// total of 3 attempts with linear backoff (1s before attempt 2, 2s before 3).
func httpGet(url string) ([]byte, error) {
	var lastErr error
	for i := 0; i < 3; i++ {
		if i > 0 {
			time.Sleep(time.Duration(i) * time.Second)
		}
		resp, err := httpClient.Get(url)
		if err != nil {
			lastErr = err
			if isRetryable(err) {
				continue
			}
			return nil, err // non-retryable: return immediately
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			if isRetryable(err) {
				continue
			}
			return nil, err
		}
		return body, nil
	}
	return nil, lastErr
}

func klineURL(secid string) string {
	return "https://push2his.eastmoney.com/api/qt/stock/kline/get?" +
		"secid=" + secid + "&fields1=f1,f2,f3,f4,f5,f6&fields2=f51,f52,f53,f54,f55,f56,f57" +
		"&klt=101&fqt=0&beg=20231201&end=20991231"
}

type apiResp struct {
	Data struct {
		Klines []string `json:"klines"`
	} `json:"data"`
}

// defaultKLineCache caches daily K-line data. Its TTL can be reconfigured at
// startup via SetKLineCacheTTL before serving traffic.
var defaultKLineCache = NewKLineCache(defaultKLineTTL)

// SetKLineCacheTTL replaces the default K-line cache with one using the given
// TTL (clamped to [1s, 24h]). Intended to be called once at startup.
func SetKLineCacheTTL(ttl time.Duration) {
	defaultKLineCache = NewKLineCache(ttl)
}

// SeedKLineCache preloads the default cache with K-line data for a secid.
// Useful for tests (to avoid network calls) and for warm-starting from a local
// store. The data is served until its TTL expires.
func SeedKLineCache(secid string, klines []KLine) {
	defaultKLineCache.Set(secid, klines)
}

// FetchKLines fetches daily K-line data from EastMoney API, serving from an
// in-memory TTL cache when a fresh entry exists.
func FetchKLines(secid string) ([]KLine, error) {
	if cached, ok := defaultKLineCache.Get(secid); ok {
		return cached, nil
	}

	body, err := httpGet(klineURL(secid))
	if err != nil {
		return nil, fmt.Errorf("HTTP请求失败: %w", err)
	}

	var r apiResp
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("解析JSON失败: %w", err)
	}

	lines := make([]KLine, 0, len(r.Data.Klines))
	for _, s := range r.Data.Klines {
		// format: "2023-12-08,0.979,0.984,0.977,0.980,50000000,49000000"
		parts := strings.Split(s, ",")
		if len(parts) < 6 {
			continue
		}
		t, err := time.Parse("2006-01-02", parts[0])
		if err != nil {
			continue
		}
		// API field order: date,open,close,high,low,volume,amount
		open, _ := strconv.ParseFloat(parts[1], 64)
		close_, _ := strconv.ParseFloat(parts[2], 64)
		high, _ := strconv.ParseFloat(parts[3], 64)
		low, _ := strconv.ParseFloat(parts[4], 64)
		vol, _ := strconv.ParseFloat(parts[5], 64)

		lines = append(lines, KLine{
			Date: t, Open: open, High: high, Low: low, Close: close_, Volume: vol,
		})
	}

	if len(lines) == 0 {
		return nil, fmt.Errorf("未获取到K线数据")
	}
	defaultKLineCache.Set(secid, lines)
	return lines, nil
}

func quoteURL(secid string) string {
	return "https://push2.eastmoney.com/api/qt/stock/get?" +
		"secid=" + secid + "&fields=f43,f44,f45,f46,f60"
}

type quoteResp struct {
	Data struct {
		F43 int `json:"f43"` // 最新价 *1000
		F44 int `json:"f44"` // 最高
		F45 int `json:"f45"` // 最低
		F46 int `json:"f46"` // 开盘
		F60 int `json:"f60"` // 昨收
	} `json:"data"`
}

func FetchQuote(secid string) (Quote, error) {
	body, err := httpGet(quoteURL(secid))
	if err != nil {
		return Quote{}, err
	}
	var r quoteResp
	if err := json.Unmarshal(body, &r); err != nil {
		return Quote{}, err
	}
	return Quote{
		Price:    float64(r.Data.F43) / 1000,
		High:     float64(r.Data.F44) / 1000,
		Low:      float64(r.Data.F45) / 1000,
		Open:     float64(r.Data.F46) / 1000,
		PreClose: float64(r.Data.F60) / 1000,
	}, nil
}
