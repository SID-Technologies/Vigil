package ipc

import (
	"context"
	"encoding/json"
	"time"

	"github.com/sid-technologies/vigil/internal/storage"
)

// RegisterOutageHandlers wires outages.list. Defaults to the last 7 days.
func RegisterOutageHandlers(s *Server, store *storage.Client) {
	s.Register("outages.list", func(ctx context.Context, params json.RawMessage) (any, *Error) {
		var p outagesListParams

		err := json.Unmarshal(params, &p)
		if err != nil {
			return nil, &Error{Code: "invalid_params", Message: err.Error()}
		}

		now := time.Now().UnixMilli()
		if p.ToMs == 0 {
			p.ToMs = now
		}

		if p.FromMs == 0 {
			p.FromMs = p.ToMs - 7*24*60*60*1000
		}

		out, err := store.Outages.Query(ctx, storage.QueryOutagesParams{
			FromMs:   p.FromMs,
			ToMs:     p.ToMs,
			Scope:    p.Scope,
			OnlyOpen: p.OnlyOpen,
		})
		if err != nil {
			return nil, &Error{Code: "internal", Message: err.Error()}
		}

		return out, nil
	})
}

type outagesListParams struct {
	FromMs   int64  `json:"from_ms,omitempty"`
	ToMs     int64  `json:"to_ms,omitempty"`
	Scope    string `json:"scope,omitempty"`
	OnlyOpen bool   `json:"only_open,omitempty"`
}
