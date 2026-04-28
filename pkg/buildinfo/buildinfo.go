// Package buildinfo exposes the binary's version, git commit, and build
// timestamp from runtime/debug.BuildInfo.
package buildinfo

import (
	"runtime/debug"

	"github.com/rs/zerolog/log"
)

// version is set by goreleaser at build-time via -ldflags.
// For dev builds it stays "main".
var version = "main"

const (
	unknown = "unknown"

	// shortCommitHashLen matches the conventional 7-char short SHA used by
	// `git log --oneline` and GitHub's commit URLs.
	shortCommitHashLen = 7
)

// Version returns the binary version (git tag for releases, "main" for dev).
func Version() string {
	return version
}

// Instrument logs the version, git commit, and timestamp once at startup.
func Instrument() {
	commit, timestamp := get()
	log.Info().
		Str("version", version).
		Str("commit", commit).
		Str("timestamp", timestamp).
		Msg("vigil sidecar build info")
}

// GitCommit returns the git commit hash, or ("", false) if unavailable.
func GitCommit() (string, bool) {
	commit, _ := get()
	if commit == unknown {
		return "", false
	}

	return commit, true
}

// get returns the git commit hash and timestamp from runtime build info.
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
			// other vcs.* keys (e.g. vcs.modified) are intentionally ignored
		}
	}

	return hash, timestamp
}
