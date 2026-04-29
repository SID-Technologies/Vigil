package ipc

import (
	"context"

	"github.com/sid-technologies/vigil/internal/probes"
	"github.com/sid-technologies/vigil/internal/storage"
)

// RegisterTargetHandlers wires targets.list/create/update/delete.
func RegisterTargetHandlers(s *Server, store *storage.Client) {
	s.Register("targets.list", bind(func(ctx context.Context, _ struct{}) ([]storage.Target, *Error) {
		out, err := store.Targets.List(ctx)
		if err != nil {
			return nil, internalErr(err)
		}

		return out, nil
	}))

	s.Register("targets.create", bind(func(ctx context.Context, p createTargetParams) (storage.Target, *Error) {
		var zero storage.Target

		if p.Label == "" || p.Host == "" || p.Kind == "" {
			return zero, &Error{Code: "invalid_params", Message: "label, host, and kind are required"}
		}

		switch probes.Kind(p.Kind) {
		case probes.KindTCP, probes.KindUDPDNS, probes.KindUDPSTUN:
			if p.Port == nil {
				return zero, &Error{Code: "invalid_params", Message: "port is required for tcp/udp_dns/udp_stun"}
			}
		case probes.KindICMP:
		default:
			return zero, &Error{Code: "invalid_params", Message: "unknown kind: " + p.Kind}
		}

		t, err := store.Targets.Create(ctx, storage.TargetRequest{
			Label: p.Label,
			Kind:  probes.Kind(p.Kind),
			Host:  p.Host,
			Port:  p.Port,
		})
		if err != nil {
			return zero, internalErr(err)
		}

		return t, nil
	}))

	s.Register("targets.update", bind(func(ctx context.Context, p updateTargetParams) (storage.Target, *Error) {
		var zero storage.Target

		if p.ID == "" {
			return zero, &Error{Code: "invalid_params", Message: "id required"}
		}

		existing, err := store.Targets.Get(ctx, p.ID)
		if err != nil {
			return zero, &Error{Code: "not_found", Message: err.Error()}
		}

		// Builtin targets only allow toggling enabled.
		if existing.IsBuiltin && (p.Host != nil || p.Port != nil) {
			return zero, &Error{Code: "builtin_immutable", Message: "builtin targets only allow toggling 'enabled'"}
		}

		t, err := store.Targets.Update(ctx, p.ID, storage.TargetUpdateRequest{
			Enabled: p.Enabled,
			Host:    p.Host,
			Port:    p.Port,
		})
		if err != nil {
			return zero, internalErr(err)
		}

		return t, nil
	}))

	s.Register("targets.delete", bind(func(ctx context.Context, p deleteTargetParams) (map[string]bool, *Error) {
		if p.ID == "" {
			return nil, &Error{Code: "invalid_params", Message: "id required"}
		}

		existing, err := store.Targets.Get(ctx, p.ID)
		if err != nil {
			return nil, &Error{Code: "not_found", Message: err.Error()}
		}

		if existing.IsBuiltin {
			return nil, &Error{Code: "builtin_immutable", Message: "builtin targets cannot be deleted; disable instead"}
		}

		err = store.Targets.Delete(ctx, p.ID)
		if err != nil {
			return nil, internalErr(err)
		}

		return map[string]bool{"ok": true}, nil
	}))
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
