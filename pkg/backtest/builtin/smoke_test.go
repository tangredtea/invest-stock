package builtin

import (
	"math"
	"testing"
	"time"

	"invest/pkg/backtest"
	"invest/pkg/data"
)

// genKlines builds a deterministic ascending-date kline series of length n with
// a mild sine-wave price pattern, for smoke testing the engine end to end.
func genKlines(n int) []data.KLine {
	out := make([]data.KLine, n)
	base := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)
	for i := 0; i < n; i++ {
		price := 10 + 2*math.Sin(float64(i)/7)
		out[i] = data.KLine{
			Date:   base.AddDate(0, 0, i),
			Open:   price,
			High:   price + 0.3,
			Low:    price - 0.3,
			Close:  price,
			Volume: 1000,
		}
	}
	return out
}

func TestEngineSmokeAllBuiltins(t *testing.T) {
	klines := genKlines(200)
	cfg := backtest.DefaultConfig()
	eng := backtest.Engine{}

	for _, name := range Names() {
		s, err := backtest.Default.New(name, nil)
		if err != nil {
			t.Fatalf("New(%q): %v", name, err)
		}
		res, err := eng.Run(klines, s, cfg)
		if err != nil {
			t.Fatalf("Run(%q): %v", name, err)
		}
		if len(res.Equity) != len(klines) {
			t.Errorf("%q: equity len = %d, want %d", name, len(res.Equity), len(klines))
		}
		if math.Abs(res.NetValue[0]-1) > 1e-9 {
			t.Errorf("%q: netValue[0] = %v, want 1", name, res.NetValue[0])
		}
		if res.Metrics.MaxDrawdown < 0 || res.Metrics.MaxDrawdown > 1 {
			t.Errorf("%q: maxDrawdown = %v, out of [0,1]", name, res.Metrics.MaxDrawdown)
		}
		if res.Metrics.WinRate < 0 || res.Metrics.WinRate > 1 {
			t.Errorf("%q: winRate = %v, out of [0,1]", name, res.Metrics.WinRate)
		}
	}
}

func TestRegistryHasNineBuiltins(t *testing.T) {
	list := backtest.Default.List()
	if len(list) != 9 {
		t.Fatalf("expected 9 builtin strategies, got %d", len(list))
	}
}
