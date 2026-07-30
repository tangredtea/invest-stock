package market

import "strings"

type Security struct {
	Code  string
	SecID string
	Name  string
}

var DefaultSecurity = Security{
	Code:  "513630",
	SecID: "1.513630",
	Name:  "港股低波红利ETF",
}

// ResolveSecID converts a stock code like "600519" to an EastMoney secid like
// "1.600519".
func ResolveSecID(code string) string {
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return ""
	}
	for _, ch := range code {
		if ch < '0' || ch > '9' {
			return ""
		}
	}
	switch code[0] {
	case '6', '5':
		return "1." + code // Shanghai
	case '0', '3', '1':
		return "0." + code // Shenzhen
	default:
		return ""
	}
}
