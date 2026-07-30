package data

import "strings"

func normalizeSecID(secid string) (string, bool) {
	secid = strings.TrimSpace(secid)
	if len(secid) != len("1.600000") || secid[1] != '.' {
		return "", false
	}
	if secid[0] != '0' && secid[0] != '1' {
		return "", false
	}
	for _, ch := range secid[2:] {
		if ch < '0' || ch > '9' {
			return "", false
		}
	}
	return secid, true
}
