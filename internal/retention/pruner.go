// Package retention deletes old samples and aggregations on a fixed cadence
// so the SQLite database doesn't grow unbounded over months of running.
//
// Retention policy (from app_config, see db/ent/schema/app_config.go):
//
//   - samples (raw)        : retention_raw_days, default 7
//   - sample_5min          : retention_5min_days, default 90
//   - sample_1h            : forever (no pruning) — historical data is the
//                            whole point of Vigil for stakeholder confronts
//   - outages              : forever
//   - wifi_samples         : same retention_raw_days as raw samples
//
// Cadence: hourly. SQLite's DELETE on an indexed timestamp column is fast;
// even pruning ~25K rows (one day's raw samples at 5/sec) takes <100ms.
//
// Implemented as DELETE WHERE ts < cutoff rather than partitioned tables —
// SQLite doesn't have native partitioning and the data sizes here don't
// warrant a manual partition scheme.
package retention

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/sid-technologies/vigil/db/ent"
	"github.com/sid-technologies/vigil/db/ent/sample"
	"github.com/sid-technologies/vigil/db/ent/sample5min"
	"github.com/sid-technologies/vigil/db/ent/wifisample"
	"github.com/sid-technologies/vigil/internal/storage"
)

// Pruner runs the retention loop. Built once at startup, runs forever until
// ctx is cancelled.
type Pruner struct {
	client *ent.Client

	// Interval between pruner wakeups. Tests can shorten; production stays
	// at 1 hour.
	Interval time.Duration
}

// New returns a Pruner with sensible defaults.
func New(client *ent.Client) *Pruner {
	return &Pruner{
		client:   client,
		Interval: 1 * time.Hour,
	}
}

// Run blocks until ctx is cancelled. Runs once on startup so a long-offline
// install with months of stale data prunes immediately rather than waiting
// an hour.
func (p *Pruner) Run(ctx context.Context) {
	p.runOnce(ctx)

	ticker := time.NewTicker(p.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("retention: shutting down")
			return
		case <-ticker.C:
			p.runOnce(ctx)
		}
	}
}

func (p *Pruner) runOnce(ctx context.Context) {
	cfg, err := storage.GetAppConfig(ctx, p.client)
	if err != nil {
		log.Error().Err(err).Msg("retention: app_config read failed")
		return
	}

	now := time.Now().UnixMilli()
	rawCutoff := now - int64(cfg.RetentionRawDays)*24*60*60*1000
	fiveMinCutoff := now - int64(cfg.Retention5minDays)*24*60*60*1000

	// Raw samples.
	rawDeleted, err := p.client.Sample.Delete().
		Where(sample.TsUnixMsLT(rawCutoff)).
		Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msg("retention: raw samples delete failed")
	} else if rawDeleted > 0 {
		log.Info().Int("rows", rawDeleted).Msg("retention: pruned raw samples")
	}

	// Wi-Fi samples — same retention as raw probes.
	wifiDeleted, err := p.client.WifiSample.Delete().
		Where(wifisample.TsUnixMsLT(rawCutoff)).
		Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msg("retention: wifi samples delete failed")
	} else if wifiDeleted > 0 {
		log.Info().Int("rows", wifiDeleted).Msg("retention: pruned wifi samples")
	}

	// 5-minute aggregations.
	fiveDeleted, err := p.client.Sample5min.Delete().
		Where(sample5min.BucketStartUnixMsLT(fiveMinCutoff)).
		Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msg("retention: 5min samples delete failed")
	} else if fiveDeleted > 0 {
		log.Info().Int("rows", fiveDeleted).Msg("retention: pruned 5min buckets")
	}

	// Sample1h and Outages: forever, no pruning.
}
