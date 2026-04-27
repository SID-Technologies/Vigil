package ipc

import (
	"context"
	"encoding/json"

	"github.com/sid-technologies/vigil/db/ent"
	"github.com/sid-technologies/vigil/internal/probes"
	"github.com/sid-technologies/vigil/internal/storage"
)

// RegisterTargetHandlers wires targets.list/create/update/delete onto the
// IPC server. Callers (cmd/vigil-sidecar) pass the Ent client they opened
// at startup.
func RegisterTargetHandlers(s *Server, client *ent.Client) {
	s.Register("targets.list", func(ctx context.Context, _ json.RawMessage) (any, *Error) {
		out, err := storage.ListTargets(ctx, client)
		if err != nil {
			return nil, &Error{Code: "internal", Message: err.Error()}
		}
		return out, nil
	})

	s.Register("targets.create", func(ctx context.Context, params json.RawMessage) (any, *Error) {
		var p createTargetParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &Error{Code: "invalid_params", Message: err.Error()}
		}
		if p.Label == "" || p.Host == "" || p.Kind == "" {
			return nil, &Error{Code: "invalid_params", Message: "label, host, and kind are required"}
		}
		// Probe-kind-specific validation: TCP/UDP must have a port.
		switch probes.Kind(p.Kind) {
		case probes.KindTCP, probes.KindUDPDNS, probes.KindUDPSTUN:
			if p.Port == nil {
				return nil, &Error{Code: "invalid_params", Message: "port is required for tcp/udp_dns/udp_stun"}
			}
		case probes.KindICMP:
			// no port required
		default:
			return nil, &Error{Code: "invalid_params", Message: "unknown kind: " + p.Kind}
		}

		t, err := storage.CreateTarget(ctx, client, p.Label, probes.Kind(p.Kind), p.Host, p.Port)
		if err != nil {
			return nil, &Error{Code: "internal", Message: err.Error()}
		}
		return t, nil
	})

	s.Register("targets.update", func(ctx context.Context, params json.RawMessage) (any, *Error) {
		var p updateTargetParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &Error{Code: "invalid_params", Message: err.Error()}
		}
		if p.ID == "" {
			return nil, &Error{Code: "invalid_params", Message: "id required"}
		}

		// Builtin targets can only have `enabled` toggled, never host/port edits.
		existing, err := storage.GetTarget(ctx, client, p.ID)
		if err != nil {
			return nil, &Error{Code: "not_found", Message: err.Error()}
		}
		if existing.IsBuiltin && (p.Host != nil || p.Port != nil) {
			return nil, &Error{Code: "builtin_immutable", Message: "builtin targets only allow toggling 'enabled'"}
		}

		t, err := storage.UpdateTarget(ctx, client, p.ID, p.Enabled, p.Host, p.Port)
		if err != nil {
			return nil, &Error{Code: "internal", Message: err.Error()}
		}
		return t, nil
	})

	s.Register("targets.delete", func(ctx context.Context, params json.RawMessage) (any, *Error) {
		var p deleteTargetParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &Error{Code: "invalid_params", Message: err.Error()}
		}
		if p.ID == "" {
			return nil, &Error{Code: "invalid_params", Message: "id required"}
		}

		existing, err := storage.GetTarget(ctx, client, p.ID)
		if err != nil {
			return nil, &Error{Code: "not_found", Message: err.Error()}
		}
		if existing.IsBuiltin {
			return nil, &Error{Code: "builtin_immutable", Message: "builtin targets cannot be deleted; disable instead"}
		}

		if err := storage.DeleteTarget(ctx, client, p.ID); err != nil {
			return nil, &Error{Code: "internal", Message: err.Error()}
		}
		return map[string]bool{"ok": true}, nil
	})
}

type createTargetParams struct {
	Label string `json:"label"`
	Kind  string `json:"kind"`
	Host  string `json:"host"`
	Port  *int   `json:"port,omitempty"`
}

type updateTargetParams struct {
	ID      string  `json:"id"`
	Enabled *bool   `json:"enabled,omitempty"`
	Host    *string `json:"host,omitempty"`
	Port    *int    `json:"port,omitempty"`
}

type deleteTargetParams struct {
	ID string `json:"id"`
}
