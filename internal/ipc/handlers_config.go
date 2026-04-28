package ipc

import (
	"context"
	"encoding/json"

	"github.com/sid-technologies/vigil/db/ent"
	"github.com/sid-technologies/vigil/internal/storage"
)

// OnConfigChange is the callback fired after a successful config.update —
// the app wires it to monitor.UpdateConfig so the running probe loop and
// flusher pick up the change without a sidecar restart.
//
// Kept as a free function type rather than importing internal/monitor to
// avoid a cycle (monitor depends on ipc.Server.Emit indirectly via the
// onCycle callback path).
type OnConfigChange func(cfg storage.AppConfig)

// RegisterConfigHandlers wires config.get and config.update. When onChange
// is non-nil, it's invoked after every successful UpdateAppConfig so the
// monitor can hot-reload its in-memory config.
func RegisterConfigHandlers(s *Server, client *ent.Client, onChange OnConfigChange) {
	s.Register("config.get", func(ctx context.Context, _ json.RawMessage) (any, *Error) {
		out, err := storage.GetAppConfig(ctx, client)
		if err != nil {
			return nil, &Error{Code: "internal", Message: err.Error()}
		}

		return out, nil
	})

	s.Register("config.update", func(ctx context.Context, params json.RawMessage) (any, *Error) {
		var patch storage.AppConfigPatch

		err := json.Unmarshal(params, &patch)
		if err != nil {
			return nil, &Error{Code: "invalid_params", Message: err.Error()}
		}

		out, err := storage.UpdateAppConfig(ctx, client, patch)
		if err != nil {
			return nil, &Error{Code: "internal", Message: err.Error()}
		}

		if onChange != nil {
			onChange(out)
		}

		return out, nil
	})
}
