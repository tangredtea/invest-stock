package backtest

import "fmt"

func cloneAndValidateSpecs(specs []ParamSpec) ([]ParamSpec, error) {
	out := cloneSpecs(specs)
	seen := make(map[string]struct{}, len(out))
	for _, spec := range out {
		if _, ok := seen[spec.Name]; ok {
			return nil, fmt.Errorf("参数名称重复: %q", spec.Name)
		}
		seen[spec.Name] = struct{}{}
		if err := validateSpec(spec); err != nil {
			return nil, err
		}
		if err := validateValue(spec, spec.Default); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func cloneSpecs(specs []ParamSpec) []ParamSpec {
	if len(specs) == 0 {
		return nil
	}
	out := make([]ParamSpec, len(specs))
	for i, spec := range specs {
		out[i] = spec
		if spec.Enum != nil {
			out[i].Enum = append([]string(nil), spec.Enum...)
		}
	}
	return out
}
