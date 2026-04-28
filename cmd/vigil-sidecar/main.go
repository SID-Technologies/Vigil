// Vigil sidecar — long-running Go process spawned by the Tauri shell.
//
// Communicates with the shell via newline-delimited JSON on stdin/stdout.
// Logs go to <data-dir>/vigil.log; never to stdout (stdout is reserved for IPC).
//
// Usage (typical, when launched by Tauri):
//
//	vigil-sidecar --data-dir /Users/you/Library/Application\ Support/dev.vigil.desktop
//
// Usage (development, when run by hand):
//
//	go run ./cmd/vigil-sidecar --dev
package main

import (
	"os"

	"github.com/sid-technologies/vigil/internal/app"
)

func main() {
	os.Exit(app.Run())
}
