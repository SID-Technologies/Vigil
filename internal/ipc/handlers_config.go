package ipc

import (
	"context"

	"github.com/sid-technologies/vigil/internal/storage"
)

// OnConfigChange fires after a successful config.update. Free function type to avoid a monitor → ipc import cycle.
type OnConfigChange func(cfg storage.AppConfig)

// RegisterConfigHandlers wires config.get and config.update.
func RegisterConfigHandlers(s *Server, store *storage.Client, onChange OnConfigChange) {
	s.Register("config.get", bind(func(ctx context.Context, _ struct{}) (storage.AppConfig, *Error) {
		out, err := store.Config.Get(ctx)
		if err != nil {
			return storage.AppConfig{}, internalErr(err)
		}

		return out, nil
	}))

	s.Register("config.update", bind(func(ctx context.Context, patch storage.AppConfigPatch) (storage.AppConfig, *Error) {
		out, err := store.Config.Update(ctx, patch)
		if err != nil {
			return storage.AppConfig{}, internalErr(err)
		}

		if onChange != nil {
			onChange(out)
		}

		return out, nil
	}))
}
