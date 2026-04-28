// Package log configures the zerolog global logger for the Vigil sidecar.
//
// IMPORTANT: the sidecar reserves stdout for stdio JSON-RPC IPC with the Tauri
// shell. Logs MUST go to a file or stderr — never stdout — or they will corrupt
// the IPC stream.
package log

import (
	"io"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

// Log rotation policy. Sized for a long-running tray app — 10 MB × 5 files
// covers ~50 MB of compressed history, comfortably more than any normal
// debug session needs while staying small enough not to alarm users.
const (
	logMaxSizeMB    = 10
	logMaxBackups   = 5
	logMaxAgeDays   = 30
	logDirPerm      = 0o755
)

// InitializeLogger sets up zerolog to write to a rolling file under dataDir.
// Output goes to <dataDir>/vigil.log with size-based rotation; older logs are
// kept compressed alongside.
//
// Returns an io.Writer for callers that want to multiplex (e.g. tee to stderr
// in development).
func InitializeLogger(dataDir string) (io.Writer, error) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	err := os.MkdirAll(dataDir, logDirPerm)
	if err != nil {
		return nil, err //nolint:wrapcheck // boundary call, plain return is fine
	}

	rotator := &lumberjack.Logger{
		Filename:   filepath.Join(dataDir, "vigil.log"),
		MaxSize:    logMaxSizeMB,
		MaxBackups: logMaxBackups,
		MaxAge:     logMaxAgeDays,
		Compress:   true,
	}

	log.Logger = zerolog.New(rotator).With().Timestamp().Logger()

	return rotator, nil
}

// InitializeLoggerStderr is for tests / `go run` development where there's no
// data dir. Logs go to stderr (NEVER stdout — that's reserved for IPC).
func InitializeLoggerStderr() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	log.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
}
