// Package buildinfo exposes version, git commit, and build timestamp.
package buildinfo

import (
	"runtime/debug"

	"github.com/rs/zerolog/log"
)

// version is set at build-time via -ldflags; "main" for dev builds.
var version = "main"

const (
	unknown            = "unknown"
	shortCommitHashLen = 7
)

// Version returns the binary version.
func Version() string {
	return version
}

// Instrument logs version, commit, and timestamp once at startup.
func Instrument() {
	commit, timestamp := get()
	log.Info().
		Str("version", version).
		Str("commit", commit).
		Str("timestamp", timestamp).
		Msg("vigil sidecar build info")
}

// GitCommit returns the short commit hash, or ("", false) if unavailable.
func GitCommit() (string, bool) {
	commit, _ := get()
	if commit == unknown {
		return "", false
	}

	return commit, true
}

func get() (hash, timestamp string) { //nolint:nonamedreturns // revive's confusing-results wants named returns for same-type tuples
	hash, timestamp = unknown, unknown
	hashLen := shortCommitHashLen

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return hash, timestamp
	}

	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			if len(s.Value) < hashLen {
				hashLen = len(s.Value)
			}

			hash = s.Value[:hashLen]
		case "vcs.time":
			timestamp = s.Value
		default:
		}
	}

	return hash, timestamp
}
