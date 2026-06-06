package builtin

import "invest/pkg/backtest"

// register adds all 9 builtin strategies to the given registry (Requirements 3.1, 3.5).
func register(r *backtest.Registry) {
	// Build a probe instance per strategy to obtain its ParamSpec list.
	specsOf := func(ctor backtest.Constructor) []backtest.ParamSpec {
		s, err := ctor(map[string]backtest.ParamValue{})
		if err != nil {
			return nil
		}
		return s.Params()
	}
	regs := []struct {
		name string
		ctor backtest.Constructor
	}{
		{NameBuyHold, newBuyHold},
		{NameMACross, newMACross},
		{NameMACDCross, newMACDCross},
		{NameBollinger, newBollinger},
		{NameRSI, newRSI},
		{NameTurtle, newTurtle},
		{NameGrid, newGrid},
		{NameMultiFactor, newMultiFactor},
		{NameDCAEnhanced, newDCAT0},
	}
	for _, rg := range regs {
		_ = r.Register(rg.name, specsOf(rg.ctor), rg.ctor)
	}
}

// Names lists the 9 builtin strategy names in canonical display order.
func Names() []string {
	return []string{
		NameBuyHold, NameMACross, NameMACDCross, NameBollinger, NameRSI,
		NameTurtle, NameGrid, NameMultiFactor, NameDCAEnhanced,
	}
}

func init() {
	register(backtest.Default)
}
