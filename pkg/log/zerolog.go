// Package log configures the zerolog global logger.
//
// stdout is reserved for JSON-RPC IPC; logs go to file or stderr only.
package log

import (
	"io"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

const (
	logMaxSizeMB  = 10
	logMaxBackups = 5
	logMaxAgeDays = 30
	logDirPerm    = 0o755
)

// InitializeLogger writes to <dataDir>/vigil.log with size-based rotation.
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

// InitializeLoggerStderr routes logs to stderr (never stdout — reserved for IPC).
func InitializeLoggerStderr() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	log.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
}
