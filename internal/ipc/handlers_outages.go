package ipc

import (
	"context"
	"encoding/json"
	"time"

	"github.com/sid-technologies/vigil/db/ent"
	"github.com/sid-technologies/vigil/internal/storage"
)

// RegisterOutageHandlers wires outages.list onto the IPC server.
//
// Default window: last 7 days. Optional `scope` filter (e.g.
// "target:google_dns_icmp" or "network"). Optional `only_open` flag for the
// dashboard's "currently degraded" badge.
func RegisterOutageHandlers(s *Server, client *ent.Client) {
	s.Register("outages.list", func(ctx context.Context, params json.RawMessage) (any, *Error) {
		var p outagesListParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &Error{Code: "invalid_params", Message: err.Error()}
		}
		now := time.Now().UnixMilli()
		if p.ToMs == 0 {
			p.ToMs = now
		}
		if p.FromMs == 0 {
			p.FromMs = p.ToMs - 7*24*60*60*1000
		}
		out, err := storage.QueryOutages(ctx, client, storage.QueryOutagesParams{
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
