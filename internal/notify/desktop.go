package notify

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// Send displays a macOS desktop notification. It is a no-op on other platforms.
// Failures are intentionally ignored because notifications must not interrupt
// monitoring or request handling.
func Send(title, msg string) {
	if runtime.GOOS != "darwin" {
		return
	}
	script := fmt.Sprintf(
		"display notification %s with title %s sound name \"Glass\"",
		appleScriptString(msg),
		appleScriptString(title),
	)
	_ = exec.Command("osascript", "-e", script).Run()
}

func appleScriptString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	return `"` + s + `"`
}
