package backtest

import (
	"math"
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"

	"invest/pkg/data"
)

type turnoverRateMinCase struct {
	Turnover float64
	Rate     float64
	Min      float64
}

func (turnoverRateMinCase) Generate(r *rand.Rand, _ int) reflect.Value {
	return reflect.ValueOf(turnoverRateMinCase{
		Turnover: r.Float64() * 1e12,
		Rate:     r.Float64() * maxCommissionRate,
		Min:      r.Float64() * maxMinCommission,
	})
}

type turnoverRateCase struct {
	Turnover float64
	Rate     float64
}

func (turnoverRateCase) Generate(r *rand.Rand, _ int) reflect.Value {
	return reflect.ValueOf(turnoverRateCase{
		Turnover: r.Float64() * 1e12,
		Rate:     r.Float64() * maxStampTaxRate,
	})
}

func nearlyEqual(got, want float64) bool {
	const tolerance = 1e-9
	diff := math.Abs(got - want)
	scale := math.Max(1, math.Abs(want))
	return diff <= tolerance*scale
}

// Feature: quant-backtest-platform, Property 17: 对任意成交额 turnover>=0、手续费比例 rate∈[0,0.01]
// 与单笔最低手续费 min∈[0,1000],计算所得手续费等于 max(turnover*rate, min)。
func TestProperty17_CommissionFormula(t *testing.T) {
	f := func(tc turnoverRateMinCase) bool {
		m := CostModel{CommissionRate: tc.Rate, MinCommission: tc.Min, Tick: defaultTick}
		got := m.Commission(tc.Turnover)
		want := tc.Turnover * tc.Rate
		if want < tc.Min {
			want = tc.Min
		}
		return nearlyEqual(got, want)
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("commission formula violated: %v", err)
	}
}

// Feature: quant-backtest-platform, Property 18: 对任意成交额与印花税率 rate∈[0,0.01],买入印花税恒为 0,
// 卖出印花税等于 turnover*rate。
func TestProperty18_StampTaxSellOnly(t *testing.T) {
	f := func(tc turnoverRateCase) bool {
		m := CostModel{StampTaxRate: tc.Rate, Tick: defaultTick}
		if m.StampTax(tc.Turnover, Buy) != 0 {
			return false
		}
		return nearlyEqual(m.StampTax(tc.Turnover, Sell), tc.Turnover*tc.Rate)
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("stamp tax sell-only violated: %v", err)
	}
}

func TestCostModelRejectsInvalidNumbers(t *testing.T) {
	m := CostModel{
		CommissionRate: 0.0003,
		MinCommission:  5,
		StampTaxRate:   0.0005,
		SlippageRatio:  0.001,
		Tick:           defaultTick,
	}
	invalid := []float64{math.NaN(), math.Inf(1), math.Inf(-1), -1}
	for _, v := range invalid {
		if got := m.Commission(v); got != 0 {
			t.Fatalf("Commission(%v) = %v, want 0", v, got)
		}
		if got := m.StampTax(v, Sell); got != 0 {
			t.Fatalf("StampTax(%v, Sell) = %v, want 0", v, got)
		}
		if got := m.FillPrice(v, Buy); got != 0 {
			t.Fatalf("FillPrice(%v, Buy) = %v, want 0", v, got)
		}
	}

	invalidModels := []CostModel{
		{SlippageMode: SlippageRatio, SlippageRatio: math.Inf(1), Tick: defaultTick},
		{SlippageMode: SlippageRatio, SlippageRatio: math.MaxFloat64, Tick: defaultTick},
		{SlippageMode: SlippageTick, SlippageTicks: 1, Tick: math.NaN()},
		{StampTaxRate: math.Inf(1), Tick: defaultTick},
	}
	for _, model := range invalidModels {
		if got := model.FillPrice(10, Buy); !finiteFloat(got) {
			t.Fatalf("FillPrice with invalid model = %v, want finite value", got)
		}
		if got := model.StampTax(10, Sell); !finiteFloat(got) {
			t.Fatalf("StampTax with invalid model = %v, want finite value", got)
		}
	}
}

// Feature: quant-backtest-platform, Property 19: 对任意基准价 base>0 与任意合法滑点配置,施加滑点后
// 买入成交价 >= base、卖出成交价 <= base,且滑点幅度等于配置所定义的比例或 tick 整数倍。
func TestProperty19_SlippageDirection(t *testing.T) {
	f := func(baseRaw, ratioRaw float64, ticks uint8, useTick bool) bool {
		base := 0.5 + math.Mod(math.Abs(baseRaw), 1000)
		ratio := math.Mod(math.Abs(ratioRaw), 0.1)
		m := CostModel{SlippageRatio: ratio, Tick: 0.01, SlippageTicks: int(ticks % 50)}
		if useTick {
			m.SlippageMode = SlippageTick
		} else {
			m.SlippageMode = SlippageRatio
		}
		buy := m.FillPrice(base, Buy)
		sell := m.FillPrice(base, Sell)
		if buy < base-1e-9 || sell > base+1e-9 {
			return false
		}
		// Slippage magnitude matches the configured definition.
		var expSlip float64
		if useTick {
			expSlip = float64(int(ticks%50)) * 0.01
		} else {
			expSlip = base * ratio
		}
		if math.Abs((buy-base)-expSlip) > 1e-9 {
			return false
		}
		// Sell may be clamped at 0; only check when no clamping occurred.
		if base-expSlip >= 0 && math.Abs((base-sell)-expSlip) > 1e-9 {
			return false
		}
		return true
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("slippage direction/magnitude violated: %v", err)
	}
}

// Feature: quant-backtest-platform, Property 20: 对任意合法回测运行产生的每条 TradeRecord,
// CostTotal == Commission + StampTax + Slippage,且 Slippage == |成交价 − 基准价| × 数量。
func TestProperty20_TradeCostConsistency(t *testing.T) {
	f := func(ks klineSeq) bool {
		eng := Engine{}
		cfg := DefaultConfig()
		cfg.Cost = CostModel{CommissionRate: 0.0003, MinCommission: 5, StampTaxRate: 0.0005, SlippageRatio: 0.001, Tick: 0.01}
		s := &testStrategy{startIndex: 60, buyAmount: cfg.InitialCash / 4}
		r, err := eng.Run([]data.KLine(ks), s, cfg)
		if err != nil {
			return true
		}
		for _, tr := range r.Trades {
			expSlip := math.Abs(tr.Price-tr.BasePrice) * float64(tr.Qty)
			if math.Abs(tr.Slippage-expSlip) > 1e-6 {
				return false
			}
			if math.Abs(tr.CostTotal-(tr.Commission+tr.StampTax+tr.Slippage)) > 1e-6 {
				return false
			}
		}
		return true
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("trade cost consistency violated: %v", err)
	}
}
