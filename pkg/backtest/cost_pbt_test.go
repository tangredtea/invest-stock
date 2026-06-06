package backtest

import (
	"math"
	"testing"
	"testing/quick"

	"invest/pkg/data"
)

// Feature: quant-backtest-platform, Property 17: 对任意成交额 turnover>=0、手续费比例 rate∈[0,0.01]
// 与单笔最低手续费 min∈[0,1000],计算所得手续费等于 max(turnover*rate, min)。
func TestProperty17_CommissionFormula(t *testing.T) {
	f := func(turnoverRaw, rateRaw, minRaw float64) bool {
		turnover := math.Abs(turnoverRaw)
		rate := math.Mod(math.Abs(rateRaw), 0.01)
		minc := math.Mod(math.Abs(minRaw), 1000)
		m := CostModel{CommissionRate: rate, MinCommission: minc, Tick: 0.01}
		got := m.Commission(turnover)
		want := turnover * rate
		if want < minc {
			want = minc
		}
		return math.Abs(got-want) <= 1e-9
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("commission formula violated: %v", err)
	}
}

// Feature: quant-backtest-platform, Property 18: 对任意成交额与印花税率 rate∈[0,0.01],买入印花税恒为 0,
// 卖出印花税等于 turnover*rate。
func TestProperty18_StampTaxSellOnly(t *testing.T) {
	f := func(turnoverRaw, rateRaw float64) bool {
		turnover := math.Abs(turnoverRaw)
		rate := math.Mod(math.Abs(rateRaw), 0.01)
		m := CostModel{StampTaxRate: rate, Tick: 0.01}
		if m.StampTax(turnover, Buy) != 0 {
			return false
		}
		return math.Abs(m.StampTax(turnover, Sell)-turnover*rate) <= 1e-9
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("stamp tax sell-only violated: %v", err)
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
