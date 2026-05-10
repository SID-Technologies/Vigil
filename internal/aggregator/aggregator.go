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
	"github.com/sid-technologies/vigil/internal/runloop"
	"github.com/sid-technologies/vigil/internal/stats"
)

// Aggregator builds 1-min, 5-min, and 1-hour rollup buckets. The 1h tier
// reads from sample_5min (not raw), so it survives raw retention pruning.
// Idempotent via (bucket_start, target_label) uniqueness — race-free because
// there's exactly one Aggregator goroutine.
type Aggregator struct {
	client *ent.Client

	// Per-tier lookback window — caps each scan so a long-offline sidecar
	// catches up on first run without reading years of history.
	Lookback1minMs int64
	Lookback5minMs int64
	Lookback1hMs   int64

	Interval time.Duration
}

// New returns an Aggregator with default lookbacks and a 1-minute interval.
func New(client *ent.Client) *Aggregator {
	return &Aggregator{
		client:         client,
		Lookback1minMs: constants.DefaultLookback1MinMs,
		Lookback5minMs: constants.DefaultLookback5MinMs,
		Lookback1hMs:   constants.DefaultLookback1HMs,
		Interval:       1 * time.Minute,
	}
}

// Run blocks until ctx is canceled. Runs once on startup so a cold-booted
// dashboard has data immediately.
func (a *Aggregator) Run(ctx context.Context) {
	runloop.Every(ctx, "aggregator", a.Interval, a.RunOnce)
}

// RunOnce drives a single aggregation pass across all tiers. Exposed for
// tests; in production it's invoked by Run on each ticker tick.
func (a *Aggregator) RunOnce(ctx context.Context) {
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

// optionalStatsMutation is satisfied by every Sample*Mutation type that
// has the optional RTT/jitter fields.
type optionalStatsMutation interface {
	SetRttP50Ms(v float64)
	SetRttP95Ms(v float64)
	SetRttP99Ms(v float64)
	SetRttMaxMs(v float64)
	SetRttMeanMs(v float64)
	SetJitterMs(v float64)
}

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

// rawTier picks the destination table for runRawAggregation.
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

type rawAggKey struct {
	bucket int64
	label  string
}

func (a *Aggregator) run1min(ctx context.Context, nowMs int64) error {
	return a.runRawAggregation(ctx, nowMs, tier1min, a.Lookback1minMs)
}

func (a *Aggregator) run5min(ctx context.Context, nowMs int64) error {
	return a.runRawAggregation(ctx, nowMs, tier5min, a.Lookback5minMs)
}

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
