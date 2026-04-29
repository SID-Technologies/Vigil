package storage

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/sid-technologies/vigil/db/ent"
	"github.com/sid-technologies/vigil/db/ent/target"
	"github.com/sid-technologies/vigil/internal/constants"
	"github.com/sid-technologies/vigil/internal/probes"
	"github.com/sid-technologies/vigil/pkg/errors"

	"github.com/google/uuid"
)

// SeedClient owns first-run seeding of builtin targets and the singleton
// app_config row.
type SeedClient struct {
	client *ent.Client
}

// NewSeedClient wraps an Ent client.
func NewSeedClient(client *ent.Client) *SeedClient {
	return &SeedClient{client: client}
}

// DefaultTargets inserts the builtin probe targets only when the table is
// empty — a user who has deleted every default should not see them resurrected
// on the next boot.
func (c *SeedClient) DefaultTargets(ctx context.Context) error {
	count, err := c.client.Target.Query().Count(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to count targets")
	}

	if count > 0 {
		return nil
	}

	defaults := probes.DefaultTargets()

	bulk := make([]*ent.TargetCreate, 0, len(defaults))
	for _, t := range defaults {
		create := c.client.Target.Create().
			SetID(uuid.NewString()).
			SetLabel(t.Label).
			SetKind(target.Kind(string(t.Kind))).
			SetHost(t.Host).
			SetEnabled(true).
			SetIsBuiltin(true)
		if t.Port != nil {
			create.SetPort(*t.Port)
		}

		bulk = append(bulk, create)
	}

	_, err = c.client.Target.CreateBulk(bulk...).Save(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to seed default targets")
	}

	log.Info().Int("count", len(bulk)).Msg("storage: seeded default targets")

	return nil
}

// AppConfig inserts the singleton app_config row on first run.
func (c *SeedClient) AppConfig(ctx context.Context) error {
	exists, err := c.client.AppConfig.Query().Where().Exist(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to check app config existence")
	}

	if exists {
		return nil
	}

	_, err = c.client.AppConfig.Create().
		SetID(AppConfigSingletonID).
		SetPingIntervalSec(constants.DefaultPingIntervalSec).
		SetFlushIntervalSec(constants.DefaultFlushIntervalSec).
		SetPingTimeoutMs(constants.DefaultPingTimeoutMs).
		SetRetentionRawDays(constants.DefaultRetentionRawDays).
		SetRetention1minDays(constants.DefaultRetention1MinDays).
		SetRetention5minDays(constants.DefaultRetention5MinDays).
		SetWifiSampleEnabled(true).
		Save(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to seed app config")
	}

	log.Info().Msg("storage: seeded app_config defaults")

	return nil
}
