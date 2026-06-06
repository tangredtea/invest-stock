package main

import (
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"

	"invest/pkg/indicator"
)

// closeSeries generates a closing-price series of length >= 2 for equivalence
// properties (which require both a latest and prior position).
type closeSeries []float64

func (closeSeries) Generate(r *rand.Rand, _ int) reflect.Value {
	n := r.Intn(200) + 2 // [2, 201]
	s := make([]float64, n)
	for i := range s {
		s[i] = 0.01 + r.Float64()*999.99
	}
	return reflect.ValueOf(closeSeries(s))
}

// anyLenSeries generates a series of any length including 0 and 1, for the
// input-length robustness property.
type anyLenSeries []float64

func (anyLenSeries) Generate(r *rand.Rand, _ int) reflect.Value {
	n := r.Intn(5) // [0, 4]
	s := make([]float64, n)
	for i := range s {
		s[i] = 0.01 + r.Float64()*999.99
	}
	return reflect.ValueOf(anyLenSeries(s))
}

// Feature: security-and-performance-hardening, Property 11: 对任意收盘价序列与任意有效下标 i,
// 从 ComputeSeries(closes) 按下标 i 取出的单点指标(indicatorsAt(series, i))与 ComputeAll(closes, i)
// 的每个对应字段逐位相等。
func TestProperty11_IndicatorsAtEqualsComputeAll(t *testing.T) {
	f := func(cs closeSeries) bool {
		closes := []float64(cs)
		series := indicator.ComputeSeries(closes)
		for i := range closes {
			got := indicatorsAt(series, i)
			want := indicator.ComputeAll(closes, i)
			if got != want {
				return false
			}
		}
		return true
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("indicatorsAt must equal ComputeAll field-for-field: %v", err)
	}
}

// Feature: security-and-performance-hardening, Property 12: 对任意有效收盘价序列(有效数据点不少于 2),
// 单次计算重构后的分析路径产出的 cur、prev 指标与重构前“两次 ComputeAll”路径在相同输入下逐位相等。
func TestProperty12_RefactorEquivalentToOldPath(t *testing.T) {
	f := func(cs closeSeries) bool {
		closes := []float64(cs)
		last := len(closes) - 1

		// New single-pass path.
		series := indicator.ComputeSeries(closes)
		newCur := indicatorsAt(series, last)
		newPrev := indicatorsAt(series, last-1)

		// Old path: two ComputeAll calls.
		oldCur := indicator.ComputeAll(closes, last)
		oldPrev := indicator.ComputeAll(closes, last-1)

		return newCur == oldCur && newPrev == oldPrev
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("refactored path must match old two-ComputeAll path: %v", err)
	}
}

// Feature: security-and-performance-hardening, Property 10: 对任意长度的收盘价序列(含 0、1),分析
// 处理器都不发生运行时 panic、不以负数或越界下标访问任何序列;当有效数据点少于 2 时按“数据不足”
// 拒绝。此测试验证长度守卫判定与底层计算在任意长度下均不 panic。
func TestProperty10_AnalyzeInputLengthRobustness(t *testing.T) {
	f := func(as anyLenSeries) (ok bool) {
		defer func() {
			if recover() != nil {
				ok = false
			}
		}()
		closes := []float64(as)
		// Mirror the handler's guard: reject (no series access) when < 2.
		if len(closes) < 2 {
			return true // guard short-circuits before any indexing
		}
		// With >= 2 points, single-pass computation and indexing must not panic.
		series := indicator.ComputeSeries(closes)
		last := len(closes) - 1
		_ = indicatorsAt(series, last)
		_ = indicatorsAt(series, last-1)
		return true
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("analyze path must be robust for any input length: %v", err)
	}
}
