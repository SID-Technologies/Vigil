// Package retention prunes old samples and aggregations hourly.
//
// Retention windows live in app_config; sample_1h and outages are kept forever.
package retention

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/sid-technologies/vigil/db/ent"
	"github.com/sid-technologies/vigil/db/ent/sample"
	"github.com/sid-technologies/vigil/db/ent/sample1min"
	"github.com/sid-technologies/vigil/db/ent/sample5min"
	"github.com/sid-technologies/vigil/db/ent/wifisample"
	"github.com/sid-technologies/vigil/internal/constants"
	"github.com/sid-technologies/vigil/internal/runloop"
	"github.com/sid-technologies/vigil/internal/storage"
)

// Pruner deletes old rows on Interval. Tests can shorten Interval; production stays at 1h.
type Pruner struct {
	client   *ent.Client
	Interval time.Duration
}

// New returns a Pruner with a 1h interval.
func New(client *ent.Client) *Pruner {
	return &Pruner{
		client:   client,
		Interval: 1 * time.Hour,
	}
}

// Run blocks until ctx is canceled. Prunes once at startup so long-offline installs catch up immediately.
func (p *Pruner) Run(ctx context.Context) {
	runloop.Every(ctx, "retention", p.Interval, p.runOnce)
}

func (p *Pruner) runOnce(ctx context.Context) {
	cfg, err := storage.NewStore(p.client).GetAppConfig(ctx)
	if err != nil {
		log.Error().Err(err).Msg("retention: app_config read failed")
		return
	}

	now := time.Now().UnixMilli()
	rawCutoff := now - int64(cfg.RetentionRawDays)*constants.OneDayMs
	oneMinCutoff := now - int64(cfg.Retention1minDays)*constants.OneDayMs
	fiveMinCutoff := now - int64(cfg.Retention5minDays)*constants.OneDayMs

	rawDeleted, err := p.client.Sample.Delete().
		Where(sample.TsUnixMsLT(rawCutoff)).
		Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msg("retention: raw samples delete failed")
	} else if rawDeleted > 0 {
		log.Info().Int("rows", rawDeleted).Msg("retention: pruned raw samples")
	}

	// wifi shares the raw-sample retention window.
	wifiDeleted, err := p.client.WifiSample.Delete().
		Where(wifisample.TsUnixMsLT(rawCutoff)).
		Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msg("retention: wifi samples delete failed")
	} else if wifiDeleted > 0 {
		log.Info().Int("rows", wifiDeleted).Msg("retention: pruned wifi samples")
	}

	oneMinDeleted, err := p.client.Sample1min.Delete().
		Where(sample1min.BucketStartUnixMsLT(oneMinCutoff)).
		Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msg("retention: 1min samples delete failed")
	} else if oneMinDeleted > 0 {
		log.Info().Int("rows", oneMinDeleted).Msg("retention: pruned 1min buckets")
	}

	fiveDeleted, err := p.client.Sample5min.Delete().
		Where(sample5min.BucketStartUnixMsLT(fiveMinCutoff)).
		Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msg("retention: 5min samples delete failed")
	} else if fiveDeleted > 0 {
		log.Info().Int("rows", fiveDeleted).Msg("retention: pruned 5min buckets")
	}
}
