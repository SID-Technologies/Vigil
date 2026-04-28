package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Sample1min is one 1-minute aggregation bucket. Sits between raw samples
// (2.5s cadence, 7-day retention) and the 5-minute tier (90-day retention).
//
// Why a 1-min tier exists: charts over 1h–6h windows want more detail than
// 5-min buckets give (12 → 72 points across the window) but raw at that
// scale ships 8k–35k points across the wire. 1-min buckets land at 60–360
// points in that band, which is the sweet spot for legibility and payload.
//
// `bucket_start_unix_ms` is the start of the 1-minute window — always a
// multiple of 60_000 ms.
//
// Schema mirrors Sample5min field-for-field. Default retention is 14 days
// (long enough to cover a "what happened last week?" investigation, short
// enough that the table stays compact even at 13 targets × 1440/day).
type Sample1min struct {
	ent.Schema
}

// Fields lists the schema fields. Required by Ent's schema interface.
func (Sample1min) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("bucket_start_unix_ms").
			Comment("Unix-ms of the 1-minute window start, multiple of 60_000"),
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
func (Sample1min) Indexes() []ent.Index {
	return []ent.Index{
		// Composite uniqueness so the aggregator can do INSERT OR IGNORE for
		// idempotent backfills. Also the chart's primary read pattern.
		index.Fields("bucket_start_unix_ms", "target_label").Unique(),
		// Time-range scans across all targets.
		index.Fields("bucket_start_unix_ms"),
	}
}
