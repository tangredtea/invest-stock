package data

import (
	"errors"
	"sync"
	"testing"
	"testing/quick"
	"time"
)

// fakeClock provides a controllable time source for deterministic TTL tests.
type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (f *fakeClock) now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.t
}

func (f *fakeClock) advance(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.t = f.t.Add(d)
}

func sampleKLines() []KLine {
	return []KLine{
		{Date: time.Unix(0, 0), Open: 1, High: 2, Low: 0.5, Close: 1.5, Volume: 100},
		{Date: time.Unix(1, 0), Open: 1.5, High: 2.5, Low: 1, Close: 2, Volume: 200},
	}
}

// fetchThroughCache simulates the FetchKLines cache-first pattern against an
// arbitrary cache + counting fetcher, so cache semantics can be tested without
// real network calls.
func fetchThroughCache(c *KLineCache, secid string, fetch func() ([]KLine, error)) ([]KLine, error) {
	if cached, ok := c.Get(secid); ok {
		return cached, nil
	}
	data, err := fetch()
	if err != nil {
		return nil, err // do not populate cache on failure
	}
	c.Set(secid, data)
	return data, nil
}

// Feature: security-and-performance-hardening, Property 13: 对任意 secid 与 TTL t,在写入后 t 时间
// 窗口内对同一 secid 的重复请求只触发恰好一次外部拉取并返回相同数据;当时间推进越过 t 后的下一次
// 请求重新触发外部拉取。
func TestProperty13_CacheTTLSemantics(t *testing.T) {
	f := func(secid string, repeats uint8) bool {
		if secid == "" {
			secid = "1.600000"
		}
		clk := &fakeClock{t: time.Unix(1_700_000_000, 0)}
		cache := NewKLineCache(time.Hour)
		cache.now = clk.now

		fetchCount := 0
		fetch := func() ([]KLine, error) {
			fetchCount++
			return sampleKLines(), nil
		}

		// First request populates the cache (1 fetch).
		fetchThroughCache(cache, secid, fetch)
		// Repeated requests within TTL must hit cache (no extra fetches).
		n := int(repeats%20) + 1
		for i := 0; i < n; i++ {
			clk.advance(time.Minute) // still within the 1h TTL
			fetchThroughCache(cache, secid, fetch)
		}
		if fetchCount != 1 {
			return false
		}
		// Advance beyond TTL → next request re-fetches.
		clk.advance(2 * time.Hour)
		fetchThroughCache(cache, secid, fetch)
		return fetchCount == 2
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("cache TTL semantics violated: %v", err)
	}
}

// Feature: security-and-performance-hardening, Property 14: 对任意外部拉取失败的 secid,数据加载器
// 返回拉取失败错误,且不为该键写入缓存条目(后续请求仍会再次尝试外部拉取)。
func TestProperty14_FailedFetchDoesNotPopulateCache(t *testing.T) {
	f := func(secid string) bool {
		if secid == "" {
			secid = "1.600000"
		}
		cache := NewKLineCache(time.Hour)
		attempts := 0
		failing := func() ([]KLine, error) {
			attempts++
			return nil, errors.New("fetch failed")
		}
		if _, err := fetchThroughCache(cache, secid, failing); err == nil {
			return false // must surface error
		}
		if _, ok := cache.Get(secid); ok {
			return false // must not cache failures
		}
		// A subsequent request must attempt the fetch again.
		fetchThroughCache(cache, secid, failing)
		return attempts == 2
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("failed fetch must not populate cache: %v", err)
	}
}

// Feature: security-and-performance-hardening, Property 15: 对任意数量的并发 goroutine 以同一 secid
// 为键并发请求 K 线数据,在 Go race detector 下无数据竞争报告,且每个并发请求都获得有效数据
// (命中时返回缓存数据,未命中时返回新拉取数据)。Run with: go test -race.
func TestProperty15_CacheConcurrencySafe(t *testing.T) {
	f := func(goroutines uint8) bool {
		n := int(goroutines%32) + 2
		cache := NewKLineCache(time.Hour)
		fetch := func() ([]KLine, error) { return sampleKLines(), nil }

		var wg sync.WaitGroup
		results := make([][]KLine, n)
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				data, _ := fetchThroughCache(cache, "1.600000", fetch)
				results[idx] = data
			}(i)
		}
		wg.Wait()
		for _, r := range results {
			if len(r) != 2 {
				return false
			}
		}
		return true
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("concurrent cache access must be safe and return valid data: %v", err)
	}
}

func TestKLineCacheTTLClamping(t *testing.T) {
	if got := NewKLineCache(time.Millisecond).ttl; got != minKLineTTL {
		t.Errorf("TTL below min should clamp to %v, got %v", minKLineTTL, got)
	}
	if got := NewKLineCache(48 * time.Hour).ttl; got != maxKLineTTL {
		t.Errorf("TTL above max should clamp to %v, got %v", maxKLineTTL, got)
	}
	if got := NewKLineCache(defaultKLineTTL).ttl; got != defaultKLineTTL {
		t.Errorf("in-range TTL should be preserved, got %v", got)
	}
}
