package ipc

import (
	"context"
	"encoding/json"

	"github.com/sid-technologies/vigil/pkg/buildinfo"
)

// RegisterCoreHandlers wires methods that don't depend on storage.
func RegisterCoreHandlers(s *Server) {
	s.Register("health.check", handleHealthCheck)
}

func handleHealthCheck(_ context.Context, _ json.RawMessage) (any, *Error) {
	commit, _ := buildinfo.GitCommit()

	return HealthCheckResult{
		Status:  "ok",
		Version: buildinfo.Version(),
		Commit:  commit,
	}, nil
}
