package aggregator

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/sid-technologies/vigil/db/ent"
	"github.com/sid-technologies/vigil/db/ent/sample"
	"github.com/sid-technologies/vigil/db/ent/sample1h"
	"github.com/sid-technologies/vigil/db/ent/sample5min"
	"github.com/sid-technologies/vigil/internal/stats"
)

// Aggregator builds 5-minute and 1-hour rollup buckets on a fixed cadence.
//
// Cadence: wakes every 5 minutes. Each wakeup:
//  1. Builds 5-min buckets for any closed-and-ready (target, window) keys
//     not already present in sample_5min.
//  2. Builds 1-hour buckets for any closed-and-ready windows. 1h rollups
//     read from sample_5min, NOT from raw — by the time retention prunes
//     raw, the 5min table still has 90 days of data.
//
// Idempotency: aggregations use the (bucket_start, target_label) unique
// index. We Query first to skip existing buckets — INSERT-OR-IGNORE would
// be cleaner but Ent doesn't expose it portably. Race-free because there's
// exactly one Aggregator goroutine.
type Aggregator struct {
	client *ent.Client

	// LookbackMs caps how far back the aggregator scans on each wakeup.
	// Defaults are tuned so a sidecar that's been off for hours will catch
	// up on first run, but doesn't read years of history every cycle.
	Lookback5minMs int64
	Lookback1hMs   int64

	// Interval between wakeups. Tests can shorten this; production stays
	// at 5 minutes.
	Interval time.Duration
}

// New builds an Aggregator with sensible defaults. Tests can override the
// fields directly afterwards.
func New(client *ent.Client) *Aggregator {
	return &Aggregator{
		client:         client,
		Lookback5minMs: 24 * 60 * 60 * 1000,      // 24h of 5-min buckets
		Lookback1hMs:   7 * 24 * 60 * 60 * 1000,  // 7d of 1-hour buckets
		Interval:       5 * time.Minute,
	}
}

// Run blocks until ctx is cancelled. On startup, runs once immediately so
// the dashboard has data to query right after the sidecar boots from cold.
func (a *Aggregator) Run(ctx context.Context) {
	a.runOnce(ctx)

	ticker := time.NewTicker(a.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("aggregator: shutting down")
			return
		case <-ticker.C:
			a.runOnce(ctx)
		}
	}
}

func (a *Aggregator) runOnce(ctx context.Context) {
	now := time.Now().UnixMilli()

	if err := a.run5min(ctx, now); err != nil {
		log.Error().Err(err).Msg("aggregator: 5min run failed")
	}
	if err := a.run1h(ctx, now); err != nil {
		log.Error().Err(err).Msg("aggregator: 1h run failed")
	}
}

// run5min finds (target, 5-min-bucket) pairs in the closed range that have
// at least one raw sample but no sample_5min row, and builds them.
func (a *Aggregator) run5min(ctx context.Context, nowMs int64) error {
	oldestBucket, newestBucket := ClosedBucketRange(nowMs, FiveMinMs, a.Lookback5minMs)
	if newestBucket < oldestBucket {
		return nil
	}

	rows, err := a.client.Sample.Query().
		Where(
			sample.TsUnixMsGTE(oldestBucket),
			sample.TsUnixMsLT(newestBucket+FiveMinMs),
		).
		All(ctx)
	if err != nil {
		return err //nolint:wrapcheck
	}
	if len(rows) == 0 {
		return nil
	}

	type key struct {
		bucket int64
		label  string
	}
	groups := make(map[key][]stats.SampleInput)
	for _, r := range rows {
		bucket := FloorBucket(r.TsUnixMs, FiveMinMs)
		k := key{bucket: bucket, label: r.TargetLabel}
		groups[k] = append(groups[k], stats.SampleInput{
			TsUnixMs: r.TsUnixMs,
			Success:  r.Success,
			RTTMs:    r.RttMs,
			Error:    r.Error,
		})
	}

	written := 0
	for k, samples := range groups {
		exists, err := a.client.Sample5min.Query().
			Where(
				sample5min.BucketStartUnixMsEQ(k.bucket),
				sample5min.TargetLabelEQ(k.label),
			).
			Exist(ctx)
		if err != nil {
			return err //nolint:wrapcheck
		}
		if exists {
			continue
		}

		summary := stats.Aggregate(samples)
		create := a.client.Sample5min.Create().
			SetBucketStartUnixMs(k.bucket).
			SetTargetLabel(k.label).
			SetCount(summary.Count).
			SetSuccessCount(summary.SuccessCount).
			SetFailCount(summary.FailCount)
		setOptionalSampleStats(create.Mutation(), summary)
		if summary.Errors != nil {
			create.SetErrors(summary.Errors)
		}
		if _, err := create.Save(ctx); err != nil {
			return err //nolint:wrapcheck
		}
		written++
	}

	if written > 0 {
		log.Info().Int("buckets", written).Msg("aggregator: wrote 5min buckets")
	}
	return nil
}

// run1h rolls up sample_5min into sample_1h. Reads only from sample_5min
// (which retains 90d) so this still works after raw is pruned.
func (a *Aggregator) run1h(ctx context.Context, nowMs int64) error {
	oldestBucket, newestBucket := ClosedBucketRange(nowMs, OneHourMs, a.Lookback1hMs)
	if newestBucket < oldestBucket {
		return nil
	}

	rows, err := a.client.Sample5min.Query().
		Where(
			sample5min.BucketStartUnixMsGTE(oldestBucket),
			sample5min.BucketStartUnixMsLT(newestBucket+OneHourMs),
		).
		All(ctx)
	if err != nil {
		return err //nolint:wrapcheck
	}
	if len(rows) == 0 {
		return nil
	}

	type key struct {
		bucket int64
		label  string
	}
	groups := make(map[key][]stats.BucketSummary)
	for _, r := range rows {
		hourBucket := FloorBucket(r.BucketStartUnixMs, OneHourMs)
		k := key{bucket: hourBucket, label: r.TargetLabel}
		summary := stats.BucketSummary{
			Count:        r.Count,
			SuccessCount: r.SuccessCount,
			FailCount:    r.FailCount,
			P50Ms:        r.RttP50Ms,
			P95Ms:        r.RttP95Ms,
			P99Ms:        r.RttP99Ms,
			MaxMs:        r.RttMaxMs,
			MeanMs:       r.RttMeanMs,
			JitterMs:     r.JitterMs,
			Errors:       r.Errors,
		}
		groups[k] = append(groups[k], summary)
	}

	written := 0
	for k, children := range groups {
		exists, err := a.client.Sample1h.Query().
			Where(
				sample1h.BucketStartUnixMsEQ(k.bucket),
				sample1h.TargetLabelEQ(k.label),
			).
			Exist(ctx)
		if err != nil {
			return err //nolint:wrapcheck
		}
		if exists {
			continue
		}

		summary := stats.AggregateFromBuckets(children)
		create := a.client.Sample1h.Create().
			SetBucketStartUnixMs(k.bucket).
			SetTargetLabel(k.label).
			SetCount(summary.Count).
			SetSuccessCount(summary.SuccessCount).
			SetFailCount(summary.FailCount)
		setOptional1hStats(create.Mutation(), summary)
		if summary.Errors != nil {
			create.SetErrors(summary.Errors)
		}
		if _, err := create.Save(ctx); err != nil {
			return err //nolint:wrapcheck
		}
		written++
	}

	if written > 0 {
		log.Info().Int("buckets", written).Msg("aggregator: wrote 1h buckets")
	}
	return nil
}

// setOptionalSampleStats applies all the *optional* fields. Pulled out to
// avoid duplicating ~12 lines of nil-checks between 5min and 1h paths.
//
// Uses the Mutation type directly rather than the typed builder because the
// 5min and 1h mutations are different generated types — generics would let
// us share more, but this is fine and explicit.
func setOptionalSampleStats(m *ent.Sample5minMutation, s stats.BucketSummary) {
	if s.P50Ms != nil {
		m.SetRttP50Ms(*s.P50Ms)
	}
	if s.P95Ms != nil {
		m.SetRttP95Ms(*s.P95Ms)
	}
	if s.P99Ms != nil {
		m.SetRttP99Ms(*s.P99Ms)
	}
	if s.MaxMs != nil {
		m.SetRttMaxMs(*s.MaxMs)
	}
	if s.MeanMs != nil {
		m.SetRttMeanMs(*s.MeanMs)
	}
	if s.JitterMs != nil {
		m.SetJitterMs(*s.JitterMs)
	}
}

func setOptional1hStats(m *ent.Sample1hMutation, s stats.BucketSummary) {
	if s.P50Ms != nil {
		m.SetRttP50Ms(*s.P50Ms)
	}
	if s.P95Ms != nil {
		m.SetRttP95Ms(*s.P95Ms)
	}
	if s.P99Ms != nil {
		m.SetRttP99Ms(*s.P99Ms)
	}
	if s.MaxMs != nil {
		m.SetRttMaxMs(*s.MaxMs)
	}
	if s.MeanMs != nil {
		m.SetRttMeanMs(*s.MeanMs)
	}
	if s.JitterMs != nil {
		m.SetJitterMs(*s.JitterMs)
	}
}
