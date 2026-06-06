package data

import (
	"sync"
	"time"
)

const (
	minKLineTTL     = 1 * time.Second
	maxKLineTTL     = 24 * time.Hour
	defaultKLineTTL = 24 * time.Hour
)

type klineCacheEntry struct {
	data      []KLine
	expiresAt time.Time
}

// KLineCache is a concurrency-safe in-memory TTL cache keyed by secid.
// Daily K-line data changes at most once per day, so caching avoids
// redundant external API calls.
type KLineCache struct {
	mu      sync.RWMutex
	ttl     time.Duration
	entries map[string]klineCacheEntry
	now     func() time.Time // injectable clock for deterministic tests
}

// NewKLineCache creates a cache with the given TTL, clamped to [1s, 24h].
func NewKLineCache(ttl time.Duration) *KLineCache {
	if ttl < minKLineTTL {
		ttl = minKLineTTL
	}
	if ttl > maxKLineTTL {
		ttl = maxKLineTTL
	}
	return &KLineCache{
		ttl:     ttl,
		entries: make(map[string]klineCacheEntry),
		now:     time.Now,
	}
}

// Get returns the cached data for secid if a non-expired entry exists.
// The returned slice is a copy to avoid aliasing the cached data.
func (c *KLineCache) Get(secid string) ([]KLine, bool) {
	c.mu.RLock()
	entry, ok := c.entries[secid]
	c.mu.RUnlock()
	if !ok || !c.now().Before(entry.expiresAt) {
		return nil, false
	}
	out := make([]KLine, len(entry.data))
	copy(out, entry.data)
	return out, true
}

// Set stores data for secid with the configured TTL. A copy of the slice is
// stored to avoid aliasing with the caller's data.
func (c *KLineCache) Set(secid string, data []KLine) {
	stored := make([]KLine, len(data))
	copy(stored, data)
	c.mu.Lock()
	c.entries[secid] = klineCacheEntry{
		data:      stored,
		expiresAt: c.now().Add(c.ttl),
	}
	c.mu.Unlock()
}
