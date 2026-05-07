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
	store    *storage.Client
	Interval time.Duration
}

// New returns a Pruner with a 1h interval.
func New(client *ent.Client, store *storage.Client) *Pruner {
	return &Pruner{
		client:   client,
		store:    store,
		Interval: 1 * time.Hour,
	}
}

// Run blocks until ctx is canceled. Prunes once at startup so long-offline installs catch up immediately.
func (p *Pruner) Run(ctx context.Context) {
	runloop.Every(ctx, "retention", p.Interval, p.runOnce)
}

func (p *Pruner) runOnce(ctx context.Context) {
	cfg, err := p.store.Config.Get(ctx)
	if err != nil {
		log.Error().Err(err).Msg("retention: app_config read failed")
		return
	}

	now := time.Now().UnixMilli()
	rawCutoff := now - int64(cfg.RetentionRawDays)*constants.OneDayMs
	oneMinCutoff := now - int64(cfg.Retention1minDays)*constants.OneDayMs
	fiveMinCutoff := now - int64(cfg.Retention5minDays)*constants.OneDayMs

	// Each prune is independent — one failure shouldn't stop the others.
	// wifi shares the raw-sample retention window.
	rawDeleted, rawErr := p.client.Sample.Delete().
		Where(sample.TsUnixMsLT(rawCutoff)).Exec(ctx)
	logPrune("raw samples", rawDeleted, rawErr)

	wifiDeleted, wifiErr := p.client.WifiSample.Delete().
		Where(wifisample.TsUnixMsLT(rawCutoff)).Exec(ctx)
	logPrune("wifi samples", wifiDeleted, wifiErr)

	oneMinDeleted, oneMinErr := p.client.Sample1min.Delete().
		Where(sample1min.BucketStartUnixMsLT(oneMinCutoff)).Exec(ctx)
	logPrune("1min buckets", oneMinDeleted, oneMinErr)

	fiveDeleted, fiveErr := p.client.Sample5min.Delete().
		Where(sample5min.BucketStartUnixMsLT(fiveMinCutoff)).Exec(ctx)
	logPrune("5min buckets", fiveDeleted, fiveErr)
}

// logPrune emits the right zerolog line for a delete result.
// No-op on zero rows so quiet hours don't spam the log file.
func logPrune(table string, rows int, err error) {
	if err != nil {
		log.Error().Err(err).Msgf("retention: %s delete failed", table)
		return
	}

	if rows == 0 {
		return
	}

	log.Info().Int("rows", rows).Msgf("retention: pruned %s", table)
}
