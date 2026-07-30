package backtest

import "fmt"

// ApplyDefaults merges user-provided params with declared defaults and validates:
//   - missing params get their default (Requirement 1.6)
//   - out-of-range -> error naming the param (Requirement 1.7)
//   - unknown name or type mismatch -> error (Requirement 1.8)
//
// Returns a validated complete param map; on any error returns nil + error and
// never constructs an instance.
func ApplyDefaults(specs []ParamSpec, in map[string]ParamValue) (map[string]ParamValue, error) {
	known := make(map[string]ParamSpec, len(specs))
	for _, s := range specs {
		if err := validateSpec(s); err != nil {
			return nil, err
		}
		known[s.Name] = s
	}
	// Reject unknown parameter names (Requirement 1.8).
	for name := range in {
		if _, ok := known[name]; !ok {
			return nil, fmt.Errorf("未知参数: %q", name)
		}
	}

	out := make(map[string]ParamValue, len(specs))
	for _, s := range specs {
		v, provided := in[s.Name]
		if !provided {
			if err := validateValue(s, s.Default); err != nil {
				return nil, err
			}
			out[s.Name] = s.Default // Requirement 1.6
			continue
		}
		if v.Type != s.Type {
			return nil, fmt.Errorf("参数 %q 类型不匹配: 期望 %s, 实际 %s", s.Name, s.Type, v.Type)
		}
		if err := validateValue(s, v); err != nil {
			return nil, err
		}
		out[s.Name] = v
	}
	return out, nil
}
