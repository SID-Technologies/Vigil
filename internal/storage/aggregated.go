package storage

import (
	"context"

	"github.com/sid-technologies/vigil/db/ent"
	"github.com/sid-technologies/vigil/db/ent/sample1h"
	"github.com/sid-technologies/vigil/db/ent/sample1min"
	"github.com/sid-technologies/vigil/db/ent/sample5min"
)

// AggregatedRow is the IPC-shape projection of a Sample1min, Sample5min, or
// Sample1h row. One Go type for all three tiers — granularity is communicated
// by the response wrapper, not by the row itself.
type AggregatedRow struct {
	BucketStartUnixMs int64          `json:"bucket_start_unix_ms"`
	TargetLabel       string         `json:"target_label"`
	Count             int            `json:"count"`
	SuccessCount      int            `json:"success_count"`
	FailCount         int            `json:"fail_count"`
	RTTP50Ms          *float64       `json:"rtt_p50_ms,omitempty"`
	RTTP95Ms          *float64       `json:"rtt_p95_ms,omitempty"`
	RTTP99Ms          *float64       `json:"rtt_p99_ms,omitempty"`
	RTTMaxMs          *float64       `json:"rtt_max_ms,omitempty"`
	RTTMeanMs         *float64       `json:"rtt_mean_ms,omitempty"`
	JitterMs          *float64       `json:"jitter_ms,omitempty"`
	Errors            map[string]int `json:"errors,omitempty"`
}

// QueryAggregatedParams scopes a sample_1min, sample_5min, or sample_1h read.
type QueryAggregatedParams struct {
	FromMs       int64
	ToMs         int64
	TargetLabels []string
}

// Query1minSamples reads from sample_1min with optional target filter.
func Query1minSamples(ctx context.Context, client *ent.Client, p QueryAggregatedParams) ([]AggregatedRow, error) {
	q := client.Sample1min.Query().
		Where(sample1min.BucketStartUnixMsGTE(p.FromMs), sample1min.BucketStartUnixMsLTE(p.ToMs)).
		Order(ent.Asc(sample1min.FieldBucketStartUnixMs))
	if len(p.TargetLabels) > 0 {
		q = q.Where(sample1min.TargetLabelIn(p.TargetLabels...))
	}

	rows, err := q.All(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // wrapped at IPC boundary
	}

	return projectAggRows(rows), nil
}

// Query5minSamples reads from sample_5min with optional target filter.
func Query5minSamples(ctx context.Context, client *ent.Client, p QueryAggregatedParams) ([]AggregatedRow, error) {
	q := client.Sample5min.Query().
		Where(sample5min.BucketStartUnixMsGTE(p.FromMs), sample5min.BucketStartUnixMsLTE(p.ToMs)).
		Order(ent.Asc(sample5min.FieldBucketStartUnixMs))
	if len(p.TargetLabels) > 0 {
		q = q.Where(sample5min.TargetLabelIn(p.TargetLabels...))
	}

	rows, err := q.All(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // wrapped at IPC boundary
	}

	return projectAggRows(rows), nil
}

// Query1hSamples reads from sample_1h with optional target filter.
func Query1hSamples(ctx context.Context, client *ent.Client, p QueryAggregatedParams) ([]AggregatedRow, error) {
	q := client.Sample1h.Query().
		Where(sample1h.BucketStartUnixMsGTE(p.FromMs), sample1h.BucketStartUnixMsLTE(p.ToMs)).
		Order(ent.Asc(sample1h.FieldBucketStartUnixMs))
	if len(p.TargetLabels) > 0 {
		q = q.Where(sample1h.TargetLabelIn(p.TargetLabels...))
	}

	rows, err := q.All(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // wrapped at IPC boundary
	}

	return projectAggRows(rows), nil
}

// projectAggRows turns any of the three Ent rollup row types into the shared
// IPC AggregatedRow. The type switch is the only practical way to share field
// access — Ent generates distinct row types per table with no shared interface.
func projectAggRows(rows any) []AggregatedRow {
	switch rs := rows.(type) {
	case []*ent.Sample1min:
		out := make([]AggregatedRow, len(rs))
		for i, r := range rs {
			out[i] = AggregatedRow{
				BucketStartUnixMs: r.BucketStartUnixMs, TargetLabel: r.TargetLabel,
				Count: r.Count, SuccessCount: r.SuccessCount, FailCount: r.FailCount,
				RTTP50Ms: r.RttP50Ms, RTTP95Ms: r.RttP95Ms, RTTP99Ms: r.RttP99Ms,
				RTTMaxMs: r.RttMaxMs, RTTMeanMs: r.RttMeanMs, JitterMs: r.JitterMs,
				Errors: r.Errors,
			}
		}

		return out
	case []*ent.Sample5min:
		out := make([]AggregatedRow, len(rs))
		for i, r := range rs {
			out[i] = AggregatedRow{
				BucketStartUnixMs: r.BucketStartUnixMs, TargetLabel: r.TargetLabel,
				Count: r.Count, SuccessCount: r.SuccessCount, FailCount: r.FailCount,
				RTTP50Ms: r.RttP50Ms, RTTP95Ms: r.RttP95Ms, RTTP99Ms: r.RttP99Ms,
				RTTMaxMs: r.RttMaxMs, RTTMeanMs: r.RttMeanMs, JitterMs: r.JitterMs,
				Errors: r.Errors,
			}
		}

		return out
	case []*ent.Sample1h:
		out := make([]AggregatedRow, len(rs))
		for i, r := range rs {
			out[i] = AggregatedRow{
				BucketStartUnixMs: r.BucketStartUnixMs, TargetLabel: r.TargetLabel,
				Count: r.Count, SuccessCount: r.SuccessCount, FailCount: r.FailCount,
				RTTP50Ms: r.RttP50Ms, RTTP95Ms: r.RttP95Ms, RTTP99Ms: r.RttP99Ms,
				RTTMaxMs: r.RttMaxMs, RTTMeanMs: r.RttMeanMs, JitterMs: r.JitterMs,
				Errors: r.Errors,
			}
		}

		return out
	}

	return nil
}
