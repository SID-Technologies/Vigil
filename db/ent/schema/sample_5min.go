package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Sample5min is one 5-minute aggregation bucket per (target, time-window).
// The aggregator goroutine builds these from raw Sample rows, then prunes
// the raw rows after retention_raw_days.
//
// `bucket_start_unix_ms` is the start of the 5-minute window — always a
// multiple of 5 * 60 * 1000 ms (300_000 ms). Buckets are non-overlapping
// and contiguous; an empty bucket simply has no row.
//
// Stats fields (rtt_p50_ms, etc.) are nullable because a bucket with all
// failures has no successful RTT samples to compute percentiles from.
//
// errors_json maps error codes ("timeout", "host_unreachable", ...) to the
// count of occurrences within this bucket. Empty map / null when there were
// no failures.
type Sample5min struct {
	ent.Schema
}

// Fields lists the schema fields. Required by Ent's schema interface.
func (Sample5min) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("bucket_start_unix_ms").
			Comment("Unix-ms of the 5-minute window start, multiple of 300_000"),
		field.String("target_label").
			NotEmpty(),
		field.Int("count").
			Comment("Total samples in the bucket"),
		field.Int("success_count"),
		field.Int("fail_count"),
		field.Float("rtt_p50_ms").Optional().Nillable(),
		field.Float("rtt_p95_ms").Optional().Nillable(),
		field.Float("rtt_p99_ms").Optional().Nillable(),
		field.Float("rtt_max_ms").Optional().Nillable(),
		field.Float("rtt_mean_ms").Optional().Nillable(),
		field.Float("jitter_ms").Optional().Nillable().
			Comment("RFC 3550-style jitter: mean abs delta of consecutive RTTs in this bucket"),
		field.JSON("errors", map[string]int{}).Optional().
			Comment("Error code → count for failed samples in this bucket"),
	}
}

// Indexes lists the schema indexes. Required by Ent's schema interface.
func (Sample5min) Indexes() []ent.Index {
	return []ent.Index{
		// Composite uniqueness so the aggregator can do INSERT OR IGNORE for
		// idempotent backfills. Also the chart's primary read pattern.
		index.Fields("bucket_start_unix_ms", "target_label").Unique(),
		// Time-range scans across all targets.
		index.Fields("bucket_start_unix_ms"),
	}
}
