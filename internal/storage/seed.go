// Package storage holds the application's persistence layer above Ent.
//
// Each .go file groups related queries (targets.go, samples.go, etc.) so the
// IPC handlers can call high-level methods like targets.List() rather than
// composing Ent query builders inline.
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


// SeedDefaultTargets inserts the 12 builtin probe targets if and only if the
// targets table is empty. Idempotent — safe to call on every startup.
//
// Distinguishing "first run" from "user deleted everything" is intentional:
// the user is allowed to remove all defaults and use only their own probe
// list. We don't re-add defaults on subsequent boots.
func SeedDefaultTargets(ctx context.Context, client *ent.Client) error {
	count, err := client.Target.Query().Count(ctx)
	if err != nil {
		return err //nolint:wrapcheck // wrapped at IPC boundary
	}

	if count > 0 {
		return nil
	}

	defaults := probes.DefaultTargets()

	bulk := make([]*ent.TargetCreate, 0, len(defaults))
	for _, t := range defaults {
		c := client.Target.Create().
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

	_, err = client.Target.CreateBulk(bulk...).Save(ctx)
	if err != nil {
		return err //nolint:wrapcheck // wrapped at IPC boundary
	}

	log.Info().Int("count", len(bulk)).Msg("storage: seeded default targets")

	return nil
}

// SeedAppConfig inserts the singleton app_config row with default values if
// it doesn't exist. Defaults match the legacy Python tool's CLI defaults so
// behavior is identical out of the box.
func SeedAppConfig(ctx context.Context, client *ent.Client) error {
	exists, err := client.AppConfig.Query().Where().Exist(ctx)
	if err != nil {
		return err //nolint:wrapcheck // wrapped at IPC boundary
	}

	if exists {
		return nil
	}

	_, err = client.AppConfig.Create().
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
