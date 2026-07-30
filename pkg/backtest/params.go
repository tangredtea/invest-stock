package backtest

// ParamType is the set of predefined parameter types (Requirement 1.5).
type ParamType int

const (
	ParamInt ParamType = iota
	ParamFloat
	ParamEnum
	ParamBool
)

func (t ParamType) String() string {
	switch t {
	case ParamInt:
		return "int"
	case ParamFloat:
		return "float"
	case ParamEnum:
		return "enum"
	case ParamBool:
		return "bool"
	default:
		return "unknown"
	}
}

// ParamValue is a tagged parameter value (avoids interface{} coercion issues).
type ParamValue struct {
	Type  ParamType `json:"type"`
	Int   int       `json:"int,omitempty"`
	Float float64   `json:"float,omitempty"`
	Str   string    `json:"str,omitempty"`
	Bool  bool      `json:"bool,omitempty"`
}

func IntVal(v int) ParamValue       { return ParamValue{Type: ParamInt, Int: v} }
func FloatVal(v float64) ParamValue { return ParamValue{Type: ParamFloat, Float: v} }
func EnumVal(v string) ParamValue   { return ParamValue{Type: ParamEnum, Str: v} }
func BoolVal(v bool) ParamValue     { return ParamValue{Type: ParamBool, Bool: v} }

// ParamSpec declares a single parameter (Requirement 1.5).
type ParamSpec struct {
	Name    string     `json:"name"` // unique and non-empty
	Type    ParamType  `json:"type"`
	Default ParamValue `json:"default"`        // must lie within range
	Min     float64    `json:"min"`            // inclusive lower bound for Int/Float
	Max     float64    `json:"max"`            // inclusive upper bound for Int/Float
	Enum    []string   `json:"enum,omitempty"` // allowed values for Enum
	Desc    string     `json:"desc,omitempty"`
}
