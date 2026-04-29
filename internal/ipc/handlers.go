package ipc

import (
	"context"
	"encoding/json"

	"github.com/sid-technologies/vigil/pkg/buildinfo"
)

// RegisterCoreHandlers wires methods that don't depend on storage.
func RegisterCoreHandlers(s *Server) {
	s.Register("health.check", bind(handleHealthCheck))
}

// bind adapts a typed handler — fn(ctx, P) (R, *Error) — to the Server's
// untyped Handler signature. Empty params unmarshal into the zero value
// of P, so handlers that don't take params can use struct{} for P.
func bind[P, R any](fn func(context.Context, P) (R, *Error)) Handler {
	return func(ctx context.Context, raw json.RawMessage) (any, *Error) {
		var p P

		if len(raw) > 0 {
			err := json.Unmarshal(raw, &p)
			if err != nil {
				return nil, &Error{Code: "invalid_params", Message: err.Error()}
			}
		}

		return fn(ctx, p)
	}
}

// internalErr wraps a Go error as an "internal" IPC error.
func internalErr(err error) *Error {
	return &Error{Code: "internal", Message: err.Error()}
}

func handleHealthCheck(_ context.Context, _ struct{}) (HealthCheckResult, *Error) {
	commit, _ := buildinfo.GitCommit()

	return HealthCheckResult{
		Status:  "ok",
		Version: buildinfo.Version(),
		Commit:  commit,
	}, nil
}
