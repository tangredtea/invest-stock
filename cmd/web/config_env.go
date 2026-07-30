package main

import (
	"os"
	"strings"
)

func envTrimmed(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}
