// Package storage is the persistence layer above Ent.
package storage

import (
	"context"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/sid-technologies/vigil/db/ent"
	"github.com/sid-technologies/vigil/db/ent/target"
	"github.com/sid-technologies/vigil/internal/constants"
	"github.com/sid-technologies/vigil/internal/probes"
)


// SeedDefaultTargets inserts the builtin probe targets only when the table is
// empty — a user who has deleted every default should not see them resurrected
// on the next boot.
func (s *Store) SeedDefaultTargets(ctx context.Context) error {
	count, err := s.client.Target.Query().Count(ctx)
	if err != nil {
		return err //nolint:wrapcheck // wrapped at IPC boundary
	}

	if count > 0 {
		return nil
	}

	defaults := probes.DefaultTargets()

	bulk := make([]*ent.TargetCreate, 0, len(defaults))
	for _, t := range defaults {
		c := s.client.Target.Create().
			SetID(uuid.NewString()).
			SetLabel(t.Label).
			SetKind(target.Kind(string(t.Kind))).
			SetHost(t.Host).
			SetEnabled(true).
			SetIsBuiltin(true)
		if t.Port != nil {
			c.SetPort(*t.Port)
		}

		bulk = append(bulk, c)
	}

	_, err = s.client.Target.CreateBulk(bulk...).Save(ctx)
	if err != nil {
		return err //nolint:wrapcheck // wrapped at IPC boundary
	}

	log.Info().Int("count", len(bulk)).Msg("storage: seeded default targets")

	return nil
}

// SeedAppConfig inserts the singleton app_config row on first run.
func (s *Store) SeedAppConfig(ctx context.Context) error {
	exists, err := s.client.AppConfig.Query().Where().Exist(ctx)
	if err != nil {
		return err //nolint:wrapcheck // wrapped at IPC boundary
	}

	if exists {
		return nil
	}

	_, err = s.client.AppConfig.Create().
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
		return err //nolint:wrapcheck // wrapped at IPC boundary
	}

	log.Info().Msg("storage: seeded app_config defaults")

	return nil
}
