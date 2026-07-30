package main

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestIsTradingTimeAtBoundaries(t *testing.T) {
	tests := []struct {
		name string
		hour int
		min  int
		want bool
	}{
		{name: "before morning open", hour: 9, min: 29},
		{name: "morning open", hour: 9, min: 30, want: true},
		{name: "morning close", hour: 11, min: 30, want: true},
		{name: "lunch break", hour: 11, min: 31},
		{name: "before afternoon open", hour: 12, min: 59},
		{name: "afternoon open", hour: 13, min: 0, want: true},
		{name: "afternoon close", hour: 15, min: 0, want: true},
		{name: "after close", hour: 15, min: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTradingTimeAt(time.Date(2024, 1, 2, tt.hour, tt.min, 0, 0, time.Local))
			if got != tt.want {
				t.Fatalf("isTradingTimeAt(%02d:%02d) = %v, want %v", tt.hour, tt.min, got, tt.want)
			}
		})
	}
}

func TestNotifyToWritesTerminalAlert(t *testing.T) {
	orig := sendDesktopNotification
	sendDesktopNotification = func(string, string) {}
	t.Cleanup(func() { sendDesktopNotification = orig })

	var buf bytes.Buffer
	notifyTo(&buf, "买入信号", "触及买入位")

	out := buf.String()
	if !strings.Contains(out, "\a") || !strings.Contains(out, "买入信号") || !strings.Contains(out, "触及买入位") {
		t.Fatalf("unexpected notification output: %q", out)
	}
}
