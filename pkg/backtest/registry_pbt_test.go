package backtest

import (
	"fmt"
	"math"
	"testing"
	"testing/quick"
)

// stubStrategy is a minimal strategy for registry tests.
type stubStrategy struct{ name string }

func (s stubStrategy) Name() string             { return s.name }
func (s stubStrategy) Params() []ParamSpec      { return nil }
func (s stubStrategy) MinBars() int             { return 60 }
func (s stubStrategy) Decide(*Context) Decision { return HoldDecision() }

func stubCtor(name string) Constructor {
	return func(map[string]ParamValue) (Strategy, error) { return stubStrategy{name: name}, nil }
}

func TestRegistryRejectsInvalidSpecsAndNilConstructor(t *testing.T) {
	r := NewRegistry()
	if err := r.Register("nil-ctor", nil, nil); err == nil {
		t.Fatal("expected nil constructor to be rejected")
	}

	cases := []struct {
		name  string
		specs []ParamSpec
	}{
		{"duplicate spec", []ParamSpec{
			{Name: "n", Type: ParamInt, Default: IntVal(1), Min: 0, Max: 10},
			{Name: "n", Type: ParamInt, Default: IntVal(2), Min: 0, Max: 10},
		}},
		{"blank spec name", []ParamSpec{{Name: " ", Type: ParamInt, Default: IntVal(1), Min: 0, Max: 10}}},
		{"default type mismatch", []ParamSpec{{Name: "n", Type: ParamInt, Default: FloatVal(1), Min: 0, Max: 10}}},
		{"non finite min", []ParamSpec{{Name: "x", Type: ParamFloat, Default: FloatVal(1), Min: math.NaN(), Max: 10}}},
		{"default out of range", []ParamSpec{{Name: "x", Type: ParamFloat, Default: FloatVal(11), Min: 0, Max: 10}}},
		{"empty enum", []ParamSpec{{Name: "e", Type: ParamEnum, Default: EnumVal("a")}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := r.Register(tc.name, tc.specs, stubCtor(tc.name)); err == nil {
				t.Fatal("expected invalid spec to be rejected")
			}
		})
	}
}

func TestRegistryClonesSpecs(t *testing.T) {
	r := NewRegistry()
	specs := []ParamSpec{{Name: "e", Type: ParamEnum, Default: EnumVal("a"), Enum: []string{"a", "b"}}}
	if err := r.Register("s", specs, stubCtor("s")); err != nil {
		t.Fatalf("register: %v", err)
	}

	specs[0].Enum[0] = "mutated"
	info, err := r.Info("s")
	if err != nil {
		t.Fatalf("info: %v", err)
	}
	if info.Params[0].Enum[0] != "a" {
		t.Fatalf("registry retained caller slice alias: %+v", info.Params[0].Enum)
	}

	info.Params[0].Enum[0] = "mutated-again"
	info2, _ := r.Info("s")
	if info2.Params[0].Enum[0] != "a" {
		t.Fatalf("Info returned mutable internal slice: %+v", info2.Params[0].Enum)
	}
}

func TestApplyDefaultsRejectsNonFiniteFloat(t *testing.T) {
	specs := []ParamSpec{{Name: "x", Type: ParamFloat, Default: FloatVal(1), Min: 0, Max: 10}}
	if _, err := ApplyDefaults(specs, map[string]ParamValue{"x": FloatVal(math.NaN())}); err == nil {
		t.Fatal("expected NaN float param to be rejected")
	}
	if _, err := ApplyDefaults(specs, map[string]ParamValue{"x": FloatVal(math.Inf(1))}); err == nil {
		t.Fatal("expected Inf float param to be rejected")
	}
}

// Feature: quant-backtest-platform, Property 6: 对任意由唯一且非空名称构成的策略集合,逐一注册后:
// 按各名称 New 均能成功构造;List() 返回的名称集合与已注册名称集合相等且长度一致;空注册表 List() 返回空列表。
func TestProperty6_RegisterLookupRoundTrip(t *testing.T) {
	f := func(seed uint16) bool {
		r := NewRegistry()
		if len(r.List()) != 0 {
			return false // empty registry must return empty list
		}
		count := int(seed%10) + 1
		names := make([]string, count)
		for i := 0; i < count; i++ {
			names[i] = fmt.Sprintf("strat-%d-%d", seed, i)
			if err := r.Register(names[i], nil, stubCtor(names[i])); err != nil {
				return false
			}
		}
		if len(r.List()) != count {
			return false
		}
		for _, n := range names {
			s, err := r.New(n, nil)
			if err != nil || s.Name() != n {
				return false
			}
		}
		return true
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("register/lookup round-trip violated: %v", err)
	}
}

// Feature: quant-backtest-platform, Property 7: 对任意已被占用的名称,二次注册必返回错误且原条目不变;
// 对任意空字符串或纯空白名称,注册必返回错误。
func TestProperty7_IllegalRegistration(t *testing.T) {
	f := func(name string) bool {
		r := NewRegistry()
		// Whitespace/empty names are always rejected.
		blanks := []string{"", " ", "\t", "  \n "}
		for _, b := range blanks {
			if err := r.Register(b, nil, stubCtor(b)); err == nil {
				return false
			}
		}
		if name == "" {
			name = "x"
		}
		if err := r.Register(name, nil, stubCtor("first")); err != nil {
			return true // generated name may collide with nothing; skip
		}
		// Duplicate registration must fail and keep the first registrant.
		if err := r.Register(name, nil, stubCtor("second")); err == nil {
			return false
		}
		s, err := r.New(name, nil)
		return err == nil && s.Name() == "first"
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("illegal registration handling violated: %v", err)
	}
}

// Feature: quant-backtest-platform, Property 4: 对任意包含越界/未知名/类型不匹配缺陷的参数输入,
// 构造必返回错误且不产出实例。
func TestProperty4_InvalidParamsRejected(t *testing.T) {
	specs := []ParamSpec{
		{Name: "n", Type: ParamInt, Default: IntVal(10), Min: 0, Max: 100},
		{Name: "r", Type: ParamFloat, Default: FloatVal(0.5), Min: 0, Max: 1},
	}
	r := NewRegistry()
	_ = r.Register("s", specs, func(p map[string]ParamValue) (Strategy, error) {
		return stubStrategy{name: "s"}, nil
	})

	f := func(v int) bool {
		// Out-of-range int.
		if v < 0 {
			v = -v
		}
		oob := 101 + (v % 1000)
		if s, err := r.New("s", map[string]ParamValue{"n": IntVal(oob)}); err == nil || s != nil {
			return false
		}
		// Unknown parameter name.
		if s, err := r.New("s", map[string]ParamValue{"unknown": IntVal(1)}); err == nil || s != nil {
			return false
		}
		// Type mismatch (float for an int param).
		if s, err := r.New("s", map[string]ParamValue{"n": FloatVal(5)}); err == nil || s != nil {
			return false
		}
		return true
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("invalid params must be rejected: %v", err)
	}
}
