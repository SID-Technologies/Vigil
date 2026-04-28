package storage

import (
	"context"

	"github.com/google/uuid"

	"github.com/sid-technologies/vigil/db/ent"
	"github.com/sid-technologies/vigil/db/ent/target"
	"github.com/sid-technologies/vigil/internal/probes"
)

// Target is the storage-layer view of a probe target. Includes the DB-level
// metadata (id, is_builtin, timestamps) that probes.Target deliberately
// omits — probes don't care about CRUD identity.
type Target struct {
	ID        string      `json:"id"`
	Label     string      `json:"label"`
	Kind      probes.Kind `json:"kind"`
	Host      string      `json:"host"`
	Port      *int        `json:"port,omitempty"`
	Enabled   bool        `json:"enabled"`
	IsBuiltin bool        `json:"is_builtin"`
}

// ListTargets returns all targets ordered by (is_builtin desc, label asc) so
// the UI shows defaults first.
func ListTargets(ctx context.Context, client *ent.Client) ([]Target, error) {
	rows, err := client.Target.Query().
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

// ListEnabledProbes returns Probe instances for every enabled target, ready
// to be installed in the monitor. Errors out if any builder fails (which
// can only happen if the DB has a corrupt enum value — won't happen in
// practice given Ent enforces enums).
func ListEnabledProbes(ctx context.Context, client *ent.Client) ([]probes.Probe, error) {
	rows, err := client.Target.Query().
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

// CreateTarget inserts a user-defined target. is_builtin is forced to false.
func CreateTarget(ctx context.Context, client *ent.Client, label string, kind probes.Kind, host string, port *int) (Target, error) {
	c := client.Target.Create().
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

// UpdateTarget mutates a target. Builtin targets cannot have host/port/kind
// changed (only enabled). The handler enforces that — this layer trusts its caller.
func UpdateTarget(ctx context.Context, client *ent.Client, id string, enabled *bool, host *string, port *int) (Target, error) {
	upd := client.Target.UpdateOneID(id)
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

// DeleteTarget removes a non-builtin target. Builtin enforcement happens at
// the handler layer (we want to surface a clear "cannot delete builtin" error code).
func DeleteTarget(ctx context.Context, client *ent.Client, id string) error {
	return client.Target.DeleteOneID(id).Exec(ctx) //nolint:wrapcheck // wrapped at IPC boundary
}

// GetTarget fetches a single target by ID.
func GetTarget(ctx context.Context, client *ent.Client, id string) (Target, error) {
	row, err := client.Target.Get(ctx, id)
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
