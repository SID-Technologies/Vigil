package ipc

import (
	"context"
	"encoding/json"

	"github.com/sid-technologies/vigil/pkg/buildinfo"
)

// RegisterCoreHandlers wires the always-available IPC methods that don't
// depend on storage or probes. As phases land, more handlers join via their
// own Register*() functions.
func RegisterCoreHandlers(s *Server) {
	s.Register("health.check", handleHealthCheck)
}

// handleHealthCheck returns sidecar version + git commit. Used by the Tauri
// shell on startup to verify the sidecar is alive and to display the version
// in the sidebar/about.
func handleHealthCheck(_ context.Context, _ json.RawMessage) (any, *Error) {
	commit, _ := buildinfo.GitCommit()

	return HealthCheckResult{
		Status:  "ok",
		Version: buildinfo.Version(),
		Commit:  commit,
	}, nil
}
