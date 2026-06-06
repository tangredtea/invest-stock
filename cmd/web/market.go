package main

import "strings"

// resolveSecID converts a stock code like "600519" to EastMoney secid like "1.600519".
func resolveSecID(code string) string {
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return ""
	}
	switch code[0] {
	case '6', '5':
		return "1." + code // 上海
	case '0', '3', '1':
		return "0." + code // 深圳
	default:
		return ""
	}
}
