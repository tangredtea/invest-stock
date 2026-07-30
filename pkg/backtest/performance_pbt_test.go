package backtest

import (
	"math"
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"
	"time"
)

// equityCurve is a generated positive equity series for metric property tests.
type equityCurve []float64

func (equityCurve) Generate(r *rand.Rand, _ int) reflect.Value {
	n := 2 + r.Intn(250)
	out := make([]float64, n)
	v := 50000 + r.Float64()*100000
	out[0] = v
	for i := 1; i < n; i++ {
		v *= 1 + (r.Float64()-0.5)*0.1
		if v < 1 {
			v = 1
		}
		out[i] = v
	}
	return reflect.ValueOf(equityCurve(out))
}

func datesFor(n int) []time.Time {
	base := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)
	ds := make([]time.Time, n)
	for i := range ds {
		ds[i] = base.AddDate(0, 0, i)
	}
	return ds
}

// Feature: quant-backtest-platform, Property 24 (metrics-level): 对任意权益曲线,最大回撤为有限实数
// 且落在 [0,1]。
func TestProperty24_MaxDrawdownBounded(t *testing.T) {
	f := func(ec equityCurve) bool {
		eq := []float64(ec)
		m, err := Calculate(eq, datesFor(len(eq)), nil, eq[0], len(eq), 0)
		if err != nil {
			return false
		}
		return finite(m.MaxDrawdown) && m.MaxDrawdown >= 0 && m.MaxDrawdown <= 1
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("max drawdown must be finite and in [0,1]: %v", err)
	}
}

// Feature: quant-backtest-platform, Property 28: 对任意合法回测运行,胜率为有限实数且落在 [0,1];
// 当不存在已平仓交易或总成交笔数为 0 时,胜率取 0。
func TestProperty28_WinRateBounded(t *testing.T) {
	f := func(ec equityCurve, sells []int8) bool {
		eq := []float64(ec)
		trades := make([]TradeRecord, 0, len(sells))
		for _, s := range sells {
			trades = append(trades, TradeRecord{Action: "sell", RealizedPL: float64(s)})
		}
		m, err := Calculate(eq, datesFor(len(eq)), trades, eq[0], len(eq), 0)
		if err != nil {
			return false
		}
		if !finite(m.WinRate) || m.WinRate < 0 || m.WinRate > 1 {
			return false
		}
		if len(trades) == 0 && m.WinRate != 0 {
			return false
		}
		return true
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("win rate must be finite and in [0,1]: %v", err)
	}
}

// Feature: quant-backtest-platform, Property 25 (degenerate): 对任意权益曲线,当有效样本数少于 2 或
// 收益方差非正时,夏普比率取 0;所有风险调整指标恒为有限实数。
func TestProperty25_SharpeFiniteAndDegenerate(t *testing.T) {
	f := func(ec equityCurve) bool {
		eq := []float64(ec)
		m, err := Calculate(eq, datesFor(len(eq)), nil, eq[0], len(eq), 0)
		if err != nil {
			return false
		}
		if !finite(m.Sharpe) || !finite(m.Sortino) || !finite(m.Calmar) || !finite(m.AnnualVolatility) {
			return false
		}
		// Constant equity => zero variance => Sharpe must be 0.
		constEq := make([]float64, 50)
		for i := range constEq {
			constEq[i] = 100000
		}
		mc, _ := Calculate(constEq, datesFor(50), nil, 100000, 50, 0)
		return mc.Sharpe == 0 && mc.AnnualVolatility == 0
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("sharpe finiteness/degeneracy violated: %v", err)
	}
}

// TestCalculateRejectsNonPositiveInitial verifies Requirement 9.1.
func TestCalculateRejectsNonPositiveInitial(t *testing.T) {
	if _, err := Calculate([]float64{1, 2}, datesFor(2), nil, 0, 1, 0); err == nil {
		t.Error("expected error for non-positive initial cash")
	}
}

func TestCalculateHandlesShortDates(t *testing.T) {
	eq := []float64{100, 120, 90}
	m, err := Calculate(eq, datesFor(1), nil, 100, 2, 0)
	if err != nil {
		t.Fatal(err)
	}
	if m.MaxDrawdown <= 0 {
		t.Fatalf("MaxDrawdown = %v, want positive drawdown", m.MaxDrawdown)
	}
	if !m.MaxDDPeakDate.IsZero() || !m.MaxDDTroughDate.IsZero() {
		t.Fatalf("drawdown dates = (%v, %v), want zero values for mismatched dates", m.MaxDDPeakDate, m.MaxDDTroughDate)
	}
}

// TestCalmarEqualsAnnualOverMDD verifies the Calmar identity on a known case.
func TestCalmarEqualsAnnualOverMDD(t *testing.T) {
	// equity rises then falls to create a known drawdown.
	eq := []float64{100, 120, 90}
	m, err := Calculate(eq, datesFor(3), nil, 100, 2, 0)
	if err != nil {
		t.Fatal(err)
	}
	if m.MaxDrawdown == 0 {
		t.Skip("no drawdown")
	}
	want := m.AnnualReturn / math.Abs(m.MaxDrawdown)
	if math.Abs(m.Calmar-want) > 1e-9 {
		t.Errorf("calmar = %v, want %v", m.Calmar, want)
	}
}
