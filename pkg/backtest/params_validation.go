package backtest

import (
	"fmt"
	"strings"
)

func validateSpec(s ParamSpec) error {
	if strings.TrimSpace(s.Name) == "" {
		return fmt.Errorf("参数名称不能为空")
	}
	if s.Default.Type != s.Type {
		return fmt.Errorf("参数 %q 默认值类型不匹配: 期望 %s, 实际 %s", s.Name, s.Type, s.Default.Type)
	}
	switch s.Type {
	case ParamInt, ParamFloat:
		if !finiteFloat(s.Min) || !finiteFloat(s.Max) || s.Min > s.Max {
			return fmt.Errorf("参数 %q 范围非法: [%g, %g]", s.Name, s.Min, s.Max)
		}
	case ParamEnum:
		if len(s.Enum) == 0 {
			return fmt.Errorf("参数 %q 枚举集合不能为空", s.Name)
		}
	case ParamBool:
		// no extra bounds
	default:
		return fmt.Errorf("参数 %q 类型非法: %s", s.Name, s.Type)
	}
	return nil
}

// validateValue checks a provided value against its spec's range/enum.
func validateValue(s ParamSpec, v ParamValue) error {
	switch s.Type {
	case ParamInt:
		f := float64(v.Int)
		if f < s.Min || f > s.Max {
			return fmt.Errorf("参数 %q 越界: %d 不在 [%g, %g]", s.Name, v.Int, s.Min, s.Max)
		}
	case ParamFloat:
		if !finiteFloat(v.Float) || v.Float < s.Min || v.Float > s.Max {
			return fmt.Errorf("参数 %q 越界: %g 不在 [%g, %g]", s.Name, v.Float, s.Min, s.Max)
		}
	case ParamEnum:
		ok := false
		for _, e := range s.Enum {
			if e == v.Str {
				ok = true
				break
			}
		}
		if !ok {
			return fmt.Errorf("参数 %q 取值非法: %q 不在枚举集合内", s.Name, v.Str)
		}
	case ParamBool:
		// any bool is valid
	}
	return nil
}
