package ipc

import (
	"context"
	"time"

	"github.com/sid-technologies/vigil/internal/storage"
)

// RegisterWifiHandlers wires wifi.list.
func RegisterWifiHandlers(s *Server, store *storage.Client) {
	s.Register("wifi.list", bind(func(ctx context.Context, p wifiListParams) ([]storage.WifiSample, *Error) {
		now := time.Now().UnixMilli()
		if p.ToMs == 0 {
			p.ToMs = now
		}

		if p.FromMs == 0 {
			p.FromMs = p.ToMs - 60*60*1000
		}

		out, err := store.Wifi.Query(ctx, p.FromMs, p.ToMs)
		if err != nil {
			return nil, internalErr(err)
		}

		return out, nil
	}))
}

type wifiListParams struct {
	FromMs int64 `json:"from_ms,omitempty"`
	ToMs   int64 `json:"to_ms,omitempty"`
}
