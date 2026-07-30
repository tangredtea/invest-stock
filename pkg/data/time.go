package data

import "time"

// DayKey truncates a timestamp to its UTC calendar day for daily-market data.
func DayKey(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
