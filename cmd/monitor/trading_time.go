package main

import "time"

func isTradingTime() bool {
	return isTradingTimeAt(time.Now())
}

func isTradingTimeAt(now time.Time) bool {
	h, m := now.Hour(), now.Minute()
	t := h*60 + m
	// A-share trading: 9:30-11:30, 13:00-15:00
	return (t >= 9*60+30 && t <= 11*60+30) || (t >= 13*60 && t <= 15*60)
}
