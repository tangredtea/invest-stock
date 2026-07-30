package main

import (
	"fmt"
	"io"
	"os"

	desktopnotify "invest/internal/notify"
)

var sendDesktopNotification = desktopnotify.Send

func notify(title, msg string) {
	notifyTo(os.Stdout, title, msg)
}

func notifyTo(w io.Writer, title, msg string) {
	// Terminal beep
	fmt.Fprint(w, "\a")
	fmt.Fprintf(w, "\n🔔 %s: %s\n\n", title, msg)

	// macOS notification center
	sendDesktopNotification(title, msg)
}
