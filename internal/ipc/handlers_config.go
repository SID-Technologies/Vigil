package ipc

import (
	"context"
	"encoding/json"

	"github.com/sid-technologies/vigil/internal/storage"
)

// OnConfigChange fires after a successful config.update. Free function type to avoid a monitor → ipc import cycle.
type OnConfigChange func(cfg storage.AppConfig)

// RegisterConfigHandlers wires config.get and config.update.
func RegisterConfigHandlers(s *Server, store *storage.Client, onChange OnConfigChange) {
	s.Register("config.get", func(ctx context.Context, _ json.RawMessage) (any, *Error) {
		out, err := store.Config.Get(ctx)
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

		out, err := store.Config.Update(ctx, patch)
		if err != nil {
			return nil, &Error{Code: "internal", Message: err.Error()}
		}

		if onChange != nil {
			onChange(out)
		}

		return out, nil
	})
}
