package aggregator

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/sid-technologies/vigil/db/ent"
	"github.com/sid-technologies/vigil/db/ent/sample"
	"github.com/sid-technologies/vigil/db/ent/sample1h"
	"github.com/sid-technologies/vigil/db/ent/sample1min"
	"github.com/sid-technologies/vigil/db/ent/sample5min"
	"github.com/sid-technologies/vigil/internal/constants"
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
	Lookback1minMs int64
	Lookback5minMs int64
	Lookback1hMs   int64

	// Interval between wakeups. Tests can shorten this; production stays
	// at 1 minute so the 1-min tier feels close to live for charts.
	Interval time.Duration
}

// New builds an Aggregator with sensible defaults. Tests can override the
// fields directly afterwards.
func New(client *ent.Client) *Aggregator {
	return &Aggregator{
		client:         client,
		Lookback1minMs: constants.DefaultLookback1MinMs,
		Lookback5minMs: constants.DefaultLookback5MinMs,
		Lookback1hMs:   constants.DefaultLookback1HMs,
		Interval:       1 * time.Minute,
	}
}

// Run blocks until ctx is canceled. On startup, runs once immediately so
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

	err := a.run1min(ctx, now)
	if err != nil {
		log.Error().Err(err).Msg("aggregator: 1min run failed")
	}

	err = a.run5min(ctx, now)
	if err != nil {
		log.Error().Err(err).Msg("aggregator: 5min run failed")
	}

	err = a.run1h(ctx, now)
	if err != nil {
		log.Error().Err(err).Msg("aggregator: 1h run failed")
	}
}

// optionalStatsMutation is satisfied by *ent.Sample1minMutation,
// *ent.Sample5minMutation, and *ent.Sample1hMutation. Ent generates these
// setters on every mutation type that has the corresponding field.
type optionalStatsMutation interface {
	SetRttP50Ms(v float64)
	SetRttP95Ms(v float64)
	SetRttP99Ms(v float64)
	SetRttMaxMs(v float64)
	SetRttMeanMs(v float64)
	SetJitterMs(v float64)
}

// setOptionalStats applies all the optional RTT/jitter fields when present.
func setOptionalStats(m optionalStatsMutation, s stats.BucketSummary) {
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

// rawTier identifies which raw-rollup destination table runRawAggregation is
// targeting. It bundles the tier-specific Ent calls behind one switch so the
// public run1min/run5min entry points stay tiny.
type rawTier int

const (
	tier1min rawTier = iota
	tier5min
)

func (t rawTier) widthMs() int64 {
	if t == tier1min {
		return OneMinMs
	}

	return FiveMinMs
}

func (t rawTier) label() string {
	if t == tier1min {
		return "1min"
	}

	return "5min"
}

// rawAggKey groups raw samples by (bucket-start, target-label).
type rawAggKey struct {
	bucket int64
	label  string
}

// run1min builds sample_1min rows from any raw samples in the closed range
// that don't yet have a corresponding bucket.
func (a *Aggregator) run1min(ctx context.Context, nowMs int64) error {
	return a.runRawAggregation(ctx, nowMs, tier1min, a.Lookback1minMs)
}

// run5min builds sample_5min rows from any raw samples in the closed range
// that don't yet have a corresponding bucket.
func (a *Aggregator) run5min(ctx context.Context, nowMs int64) error {
	return a.runRawAggregation(ctx, nowMs, tier5min, a.Lookback5minMs)
}

// runRawAggregation builds rollup buckets from raw Sample rows. Tier
// determines bucket width and the destination Ent table.
func (a *Aggregator) runRawAggregation(ctx context.Context, nowMs int64, tier rawTier, lookbackMs int64) error {
	width := tier.widthMs()

	oldestBucket, newestBucket := ClosedBucketRange(nowMs, width, lookbackMs)
	if newestBucket < oldestBucket {
		return nil
	}

	rows, err := a.client.Sample.Query().
		Where(
			sample.TsUnixMsGTE(oldestBucket),
			sample.TsUnixMsLT(newestBucket+width),
		).
		All(ctx)
	if err != nil {
		return err //nolint:wrapcheck // logged by goroutine; not surfaced to IPC
	}

	if len(rows) == 0 {
		return nil
	}

	groups := make(map[rawAggKey][]stats.SampleInput)

	for _, r := range rows {
		k := rawAggKey{bucket: FloorBucket(r.TsUnixMs, width), label: r.TargetLabel}
		groups[k] = append(groups[k], stats.SampleInput{
			TSUnixMs: r.TsUnixMs,
			Success:  r.Success,
			RTTMs:    r.RttMs,
			Error:    r.Error,
		})
	}

	written := 0

	for k, samples := range groups {
		got, err := a.rawTierExists(ctx, tier, k.bucket, k.label)
		if err != nil {
			return err
		}

		if got {
			continue
		}

		summary := stats.Aggregate(samples)

		err = a.rawTierSave(ctx, tier, k.bucket, k.label, summary)
		if err != nil {
			return err
		}

		written++
	}

	if written > 0 {
		log.Info().Int("buckets", written).Msgf("aggregator: wrote %s buckets", tier.label())
	}

	return nil
}

func (a *Aggregator) rawTierExists(ctx context.Context, tier rawTier, bucket int64, label string) (bool, error) {
	if tier == tier1min {
		got, err := a.client.Sample1min.Query().
			Where(sample1min.BucketStartUnixMsEQ(bucket), sample1min.TargetLabelEQ(label)).
			Exist(ctx)
		if err != nil {
			return false, err //nolint:wrapcheck // logged by goroutine; not surfaced to IPC
		}

		return got, nil
	}

	got, err := a.client.Sample5min.Query().
		Where(sample5min.BucketStartUnixMsEQ(bucket), sample5min.TargetLabelEQ(label)).
		Exist(ctx)
	if err != nil {
		return false, err //nolint:wrapcheck // logged by goroutine; not surfaced to IPC
	}

	return got, nil
}

func (a *Aggregator) rawTierSave(ctx context.Context, tier rawTier, bucket int64, label string, summary stats.BucketSummary) error {
	if tier == tier1min {
		create := a.client.Sample1min.Create().
			SetBucketStartUnixMs(bucket).
			SetTargetLabel(label).
			SetCount(summary.Count).
			SetSuccessCount(summary.SuccessCount).
			SetFailCount(summary.FailCount)
		setOptionalStats(create.Mutation(), summary)

		if summary.Errors != nil {
			create.SetErrors(summary.Errors)
		}

		_, err := create.Save(ctx)
		if err != nil {
			return err //nolint:wrapcheck // logged by goroutine; not surfaced to IPC
		}

		return nil
	}

	create := a.client.Sample5min.Create().
		SetBucketStartUnixMs(bucket).
		SetTargetLabel(label).
		SetCount(summary.Count).
		SetSuccessCount(summary.SuccessCount).
		SetFailCount(summary.FailCount)
	setOptionalStats(create.Mutation(), summary)

	if summary.Errors != nil {
		create.SetErrors(summary.Errors)
	}

	_, err := create.Save(ctx)
	if err != nil {
		return err //nolint:wrapcheck // logged by goroutine; not surfaced to IPC
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
		return err //nolint:wrapcheck // logged by goroutine; not surfaced to IPC
	}

	if len(rows) == 0 {
		return nil
	}

	groups := make(map[rawAggKey][]stats.BucketSummary)

	for _, r := range rows {
		k := rawAggKey{bucket: FloorBucket(r.BucketStartUnixMs, OneHourMs), label: r.TargetLabel}
		groups[k] = append(groups[k], stats.BucketSummary{
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
		})
	}

	written := 0

	for k, children := range groups {
		got, err := a.client.Sample1h.Query().
			Where(sample1h.BucketStartUnixMsEQ(k.bucket), sample1h.TargetLabelEQ(k.label)).
			Exist(ctx)
		if err != nil {
			return err //nolint:wrapcheck // logged by goroutine; not surfaced to IPC
		}

		if got {
			continue
		}

		summary := stats.AggregateFromBuckets(children)
		create := a.client.Sample1h.Create().
			SetBucketStartUnixMs(k.bucket).
			SetTargetLabel(k.label).
			SetCount(summary.Count).
			SetSuccessCount(summary.SuccessCount).
			SetFailCount(summary.FailCount)
		setOptionalStats(create.Mutation(), summary)

		if summary.Errors != nil {
			create.SetErrors(summary.Errors)
		}

		_, err = create.Save(ctx)
		if err != nil {
			return err //nolint:wrapcheck // logged by goroutine; not surfaced to IPC
		}

		written++
	}

	if written > 0 {
		log.Info().Int("buckets", written).Msg("aggregator: wrote 1h buckets")
	}

	return nil
}
