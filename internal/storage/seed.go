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
		return err //nolint:wrapcheck
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

	if _, err := client.Target.CreateBulk(bulk...).Save(ctx); err != nil {
		return err //nolint:wrapcheck
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
		return err //nolint:wrapcheck
	}
	if exists {
		return nil
	}
	_, err = client.AppConfig.Create().
		SetID(1).
		SetPingIntervalSec(2.5).
		SetFlushIntervalSec(60).
		SetPingTimeoutMs(2000).
		SetRetentionRawDays(7).
		SetRetention5minDays(90).
		SetWifiSampleEnabled(true).
		Save(ctx)
	if err != nil {
		return err //nolint:wrapcheck
	}
	log.Info().Msg("storage: seeded app_config defaults")
	return nil
}
