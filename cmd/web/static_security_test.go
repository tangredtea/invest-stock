package main

import (
	"os"
	"strings"
	"testing"
)

func TestStaticRenderScriptsKeepHTMLEscaping(t *testing.T) {
	for _, path := range []string{
		"static/js/dashboard.js",
		"static/js/backtest.js",
		"static/js/portfolio.js",
	} {
		t.Run(path, func(t *testing.T) {
			body, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read script: %v", err)
			}
			src := string(body)
			if !strings.Contains(src, "function esc") && !strings.Contains(src, "const esc") {
				t.Fatalf("%s must keep an HTML escaping helper", path)
			}
			for _, token := range []string{"replace(/&/g", "replace(/</g", "replace(/>/g"} {
				if !strings.Contains(src, token) {
					t.Fatalf("%s must keep HTML escaping helper token %q", path, token)
				}
			}
		})
	}
}
