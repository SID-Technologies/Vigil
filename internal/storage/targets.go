package storage

import (
	"context"

	"github.com/sid-technologies/vigil/db/ent"
	"github.com/sid-technologies/vigil/db/ent/target"
	"github.com/sid-technologies/vigil/internal/probes"
	"github.com/sid-technologies/vigil/pkg/errors"

	"github.com/google/uuid"
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

// TargetRequest is the input shape for Create.
type TargetRequest struct {
	Label string
	Kind  probes.Kind
	Host  string
	Port  *int
}

// TargetUpdateRequest is the input shape for Update. All fields optional;
// nil means "leave unchanged".
type TargetUpdateRequest struct {
	Enabled *bool
	Host    *string
	Port    *int
}

// TargetClient owns CRUD against the target table.
type TargetClient struct {
	client *ent.Client
}

// NewTargetClient wraps an Ent client.
func NewTargetClient(client *ent.Client) *TargetClient {
	return &TargetClient{client: client}
}

// List returns every target ordered by label.
func (c *TargetClient) List(ctx context.Context) ([]Target, error) {
	rows, err := c.client.Target.Query().
		Order(ent.Asc(target.FieldLabel)).
		All(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list targets")
	}

	out := make([]Target, 0, len(rows))
	for _, r := range rows {
		out = append(out, toTarget(r))
	}

	return out, nil
}

// ListEnabledProbes returns built Probe instances for every enabled target.
func (c *TargetClient) ListEnabledProbes(ctx context.Context) ([]probes.Probe, error) {
	rows, err := c.client.Target.Query().
		Where(target.EnabledEQ(true)).
		All(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list enabled probes")
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
			return nil, errors.Wrap(err, "failed to build probe")
		}

		out = append(out, probe)
	}

	return out, nil
}

// Create inserts a user-defined target with is_builtin=false.
func (c *TargetClient) Create(ctx context.Context, req TargetRequest) (Target, error) {
	create := c.client.Target.Create().
		SetID(uuid.NewString()).
		SetLabel(req.Label).
		SetKind(target.Kind(string(req.Kind))).
		SetHost(req.Host).
		SetEnabled(true).
		SetIsBuiltin(false)
	if req.Port != nil {
		create.SetPort(*req.Port)
	}

	row, err := create.Save(ctx)
	if err != nil {
		return Target{}, errors.Wrap(err, "failed to create target")
	}

	return toTarget(row), nil
}

// Update — handler enforces the builtin host/port/kind immutability; this
// layer trusts its caller.
func (c *TargetClient) Update(ctx context.Context, id string, req TargetUpdateRequest) (Target, error) {
	upd := c.client.Target.UpdateOneID(id)
	if req.Enabled != nil {
		upd.SetEnabled(*req.Enabled)
	}

	if req.Host != nil {
		upd.SetHost(*req.Host)
	}

	if req.Port != nil {
		upd.SetPort(*req.Port)
	}

	row, err := upd.Save(ctx)
	if err != nil {
		return Target{}, errors.Wrap(err, "failed to update target")
	}

	return toTarget(row), nil
}

// Delete — builtin guard lives at the handler so it can return a typed error.
func (c *TargetClient) Delete(ctx context.Context, id string) error {
	err := c.client.Target.DeleteOneID(id).Exec(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to delete target")
	}

	return nil
}

// Get fetches a target by id.
func (c *TargetClient) Get(ctx context.Context, id string) (Target, error) {
	row, err := c.client.Target.Get(ctx, id)
	if err != nil {
		return Target{}, errors.Wrap(err, "failed to get target")
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
