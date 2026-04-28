package storage

import (
	"context"

	"github.com/google/uuid"

	"github.com/sid-technologies/vigil/db/ent"
	"github.com/sid-technologies/vigil/db/ent/target"
	"github.com/sid-technologies/vigil/internal/probes"
)

// Target adds DB-level metadata (id, is_builtin) on top of probes.Target,
// which deliberately omits CRUD identity.
type Target struct {
	ID        string      `json:"id"`
	Label     string      `json:"label"`
	Kind      probes.Kind `json:"kind"`
	Host      string      `json:"host"`
	Port      *int        `json:"port,omitempty"`
	Enabled   bool        `json:"enabled"`
	IsBuiltin bool        `json:"is_builtin"`
}

// ListTargets returns every target ordered by label.
func (s *Store) ListTargets(ctx context.Context) ([]Target, error) {
	rows, err := s.client.Target.Query().
		Order(ent.Asc(target.FieldLabel)).
		All(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // wrapped at IPC boundary
	}

	out := make([]Target, 0, len(rows))
	for _, r := range rows {
		out = append(out, toTarget(r))
	}

	return out, nil
}

// ListEnabledProbes returns built Probe instances for every enabled target.
func (s *Store) ListEnabledProbes(ctx context.Context) ([]probes.Probe, error) {
	rows, err := s.client.Target.Query().
		Where(target.EnabledEQ(true)).
		All(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // wrapped at IPC boundary
	}

	out := make([]probes.Probe, 0, len(rows))
	for _, r := range rows {
		t := probes.Target{
			Label: r.Label,
			Kind:  probes.Kind(r.Kind),
			Host:  r.Host,
		}
		if r.Port != nil {
			t.Port = r.Port
		}

		probe, err := probes.Build(t)
		if err != nil {
			return nil, err //nolint:wrapcheck // wrapped at IPC boundary
		}

		out = append(out, probe)
	}

	return out, nil
}

// CreateTarget inserts a user-defined target with is_builtin=false.
func (s *Store) CreateTarget(ctx context.Context, label string, kind probes.Kind, host string, port *int) (Target, error) {
	c := s.client.Target.Create().
		SetID(uuid.NewString()).
		SetLabel(label).
		SetKind(target.Kind(string(kind))).
		SetHost(host).
		SetEnabled(true).
		SetIsBuiltin(false)
	if port != nil {
		c.SetPort(*port)
	}

	row, err := c.Save(ctx)
	if err != nil {
		return Target{}, err //nolint:wrapcheck // wrapped at IPC boundary
	}

	return toTarget(row), nil
}

// UpdateTarget — handler enforces the builtin host/port/kind immutability;
// this layer trusts its caller.
func (s *Store) UpdateTarget(ctx context.Context, id string, enabled *bool, host *string, port *int) (Target, error) {
	upd := s.client.Target.UpdateOneID(id)
	if enabled != nil {
		upd.SetEnabled(*enabled)
	}

	if host != nil {
		upd.SetHost(*host)
	}

	if port != nil {
		upd.SetPort(*port)
	}

	row, err := upd.Save(ctx)
	if err != nil {
		return Target{}, err //nolint:wrapcheck // wrapped at IPC boundary
	}

	return toTarget(row), nil
}

// DeleteTarget — builtin guard lives at the handler so it can return a typed error.
func (s *Store) DeleteTarget(ctx context.Context, id string) error {
	return s.client.Target.DeleteOneID(id).Exec(ctx) //nolint:wrapcheck // wrapped at IPC boundary
}

// GetTarget fetches a target by id.
func (s *Store) GetTarget(ctx context.Context, id string) (Target, error) {
	row, err := s.client.Target.Get(ctx, id)
	if err != nil {
		return Target{}, err //nolint:wrapcheck // wrapped at IPC boundary
	}

	return toTarget(row), nil
}

func toTarget(r *ent.Target) Target {
	t := Target{
		ID:        r.ID,
		Label:     r.Label,
		Kind:      probes.Kind(r.Kind),
		Host:      r.Host,
		Enabled:   r.Enabled,
		IsBuiltin: r.IsBuiltin,
	}
	if r.Port != nil {
		t.Port = r.Port
	}

	return t
}
