package backtest

import (
	"testing"
	"testing/quick"
)

// Feature: quant-backtest-platform, Property 22: 对任意启用 T+1 的回测,某根 K线的可卖数量等于"更早
// K线买入且仍持有的批次数量之和",取值范围为 [0, 当前总持仓] 且不含当根买入数量;未启用 T+1 时可卖
// 数量等于当前总持仓。
func TestProperty22_TPlus1Sellable(t *testing.T) {
	f := func(buys []uint8) bool {
		acc := newAccount(0)
		// Buy on increasing bar indices.
		for i, q := range buys {
			qty := int(q%10) + 1
			acc.addLot(qty, i, 1.0, 0)
		}
		curIdx := len(buys) // a bar strictly after all buys
		total := acc.shares()

		// Without T+1, everything is sellable.
		if acc.sellable(curIdx, false) != total {
			return false
		}
		// With T+1 at curIdx (after all buys), all lots are from earlier bars.
		if acc.sellable(curIdx, true) != total {
			return false
		}
		// With T+1 at the last buy's index, that lot is not yet sellable.
		if len(buys) > 0 {
			lastIdx := len(buys) - 1
			sellableNow := acc.sellable(lastIdx, true)
			lastQty := acc.lots[lastIdx].qty
			if sellableNow != total-lastQty {
				return false
			}
			if sellableNow < 0 || sellableNow > total {
				return false
			}
		}
		return true
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("T+1 sellable invariant violated: %v", err)
	}
}
