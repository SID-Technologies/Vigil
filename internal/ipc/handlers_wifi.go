package ipc

import (
	"context"
	"encoding/json"
	"time"

	"github.com/sid-technologies/vigil/db/ent"
	"github.com/sid-technologies/vigil/internal/storage"
)

// RegisterWifiHandlers wires wifi.list onto the IPC server.
func RegisterWifiHandlers(s *Server, client *ent.Client) {
	s.Register("wifi.list", func(ctx context.Context, params json.RawMessage) (any, *Error) {
		var p wifiListParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &Error{Code: "invalid_params", Message: err.Error()}
		}
		now := time.Now().UnixMilli()
		if p.ToMs == 0 {
			p.ToMs = now
		}
		if p.FromMs == 0 {
			p.FromMs = p.ToMs - 60*60*1000
		}
		out, err := storage.QueryWifiSamples(ctx, client, p.FromMs, p.ToMs)
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
