package main

import (
	"sync"
	"testing"
	"testing/quick"
	"time"
)

// rlClock is a controllable clock for limiter tests.
type rlClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *rlClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *rlClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

// Feature: security-and-performance-hardening, Property 6: 对任意阈值 T∈[1,100] 与窗口
// W∈[60,86400]s 配置,以及同一客户端 IP 在窗口内的连续失败序列:当累计失败次数小于 T 时限流器放行,
// 达到 T 后对后续尝试一律拒绝。
func TestProperty6_FailureThreshold(t *testing.T) {
	f := func(rawT uint8, rawW uint32) bool {
		threshold := int(rawT%100) + 1   // [1,100]
		window := time.Duration(int(rawW%86340)+60) * time.Second // [60,86400]s
		clk := &rlClock{t: time.Unix(1_700_000_000, 0)}
		l := NewLoginRateLimiter(threshold, window)
		l.now = clk.now
		ip := "10.0.0.1"

		// Below threshold: each attempt must be allowed, then we record a failure.
		for i := 0; i < l.threshold; i++ {
			if !l.Allowed(ip) {
				return false
			}
			l.RecordFailure(ip)
		}
		// At threshold: further attempts (within window) must be rejected.
		return !l.Allowed(ip)
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("failure threshold property violated: %v", err)
	}
}

// Feature: security-and-performance-hardening, Property 7: 对任意已达阈值而被限流的客户端 IP,当时间
// 推进越过 windowStart + W 后,限流器再次放行该 IP 的登录尝试。
func TestProperty7_WindowExpiryRecovery(t *testing.T) {
	f := func(rawW uint32) bool {
		window := time.Duration(int(rawW%86340)+60) * time.Second
		clk := &rlClock{t: time.Unix(1_700_000_000, 0)}
		l := NewLoginRateLimiter(3, window)
		l.now = clk.now
		ip := "10.0.0.2"

		for i := 0; i < l.threshold; i++ {
			l.RecordFailure(ip)
		}
		if l.Allowed(ip) {
			return false // should be blocked now
		}
		clk.advance(l.window + time.Second) // cross the window boundary
		return l.Allowed(ip)
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("window expiry recovery property violated: %v", err)
	}
}

// Feature: security-and-performance-hardening, Property 8: 对任意客户端 IP 在窗口内的任意失败计数
// (含已达阈值),一旦该 IP 登录成功调用 Reset,其失败计数归零且后续尝试被放行。
func TestProperty8_ResetClearsFailures(t *testing.T) {
	f := func(failures uint8) bool {
		l := NewLoginRateLimiter(3, defaultLoginWindow)
		ip := "10.0.0.3"
		n := int(failures%10) + 1
		for i := 0; i < n; i++ {
			l.RecordFailure(ip)
		}
		l.Reset(ip)
		return l.Allowed(ip)
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("reset must clear failures: %v", err)
	}
}

func TestLimiterClampsConfig(t *testing.T) {
	l := NewLoginRateLimiter(0, time.Second)
	if l.threshold != 1 {
		t.Errorf("threshold below min should clamp to 1, got %d", l.threshold)
	}
	if l.window != 60*time.Second {
		t.Errorf("window below min should clamp to 60s, got %v", l.window)
	}
	l2 := NewLoginRateLimiter(500, 100000*time.Second)
	if l2.threshold != 100 {
		t.Errorf("threshold above max should clamp to 100, got %d", l2.threshold)
	}
	if l2.window != 86400*time.Second {
		t.Errorf("window above max should clamp to 86400s, got %v", l2.window)
	}
}
