package ipc

import (
	"context"
	"encoding/json"
	"time"

	"github.com/sid-technologies/vigil/internal/storage"
)

// RegisterWifiHandlers wires wifi.list.
func RegisterWifiHandlers(s *Server, store *storage.Store) {
	s.Register("wifi.list", func(ctx context.Context, params json.RawMessage) (any, *Error) {
		var p wifiListParams

		err := json.Unmarshal(params, &p)
		if err != nil {
			return nil, &Error{Code: "invalid_params", Message: err.Error()}
		}

		now := time.Now().UnixMilli()
		if p.ToMs == 0 {
			p.ToMs = now
		}

		if p.FromMs == 0 {
			p.FromMs = p.ToMs - 60*60*1000
		}

		out, err := store.QueryWifiSamples(ctx, p.FromMs, p.ToMs)
		if err != nil {
			return nil, &Error{Code: "internal", Message: err.Error()}
		}

		return out, nil
	})
}

type wifiListParams struct {
	FromMs int64 `json:"from_ms,omitempty"`
	ToMs   int64 `json:"to_ms,omitempty"`
}
