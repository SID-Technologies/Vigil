package storage

import "github.com/sid-technologies/vigil/db/ent"

// AggregatedRow is the IPC shape shared across the 1min/5min/1h rollup tiers;
// granularity is signaled by the response wrapper, not the row.
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

// QueryAggregatedParams scopes a rollup-tier read.
type QueryAggregatedParams struct {
	FromMs       int64
	ToMs         int64
	TargetLabels []string
}

// projectAggRows collapses Ent's three distinct rollup row types (no shared
// interface generated) into the IPC AggregatedRow.
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
