package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Sample1h is one 1-hour aggregation bucket. Built by rolling up twelve
// 5-minute buckets, NOT from raw samples — far cheaper at scale and
// indistinguishable in chart fidelity for hour+ time ranges.
//
// `bucket_start_unix_ms` is always a multiple of 60 * 60 * 1000 ms
// (3_600_000 ms).
//
// 1-hour buckets are kept indefinitely (no retention pruning) so users have
// historical data for stakeholder confrontations months/years later.
//
// Schema mirrors Sample5min field-for-field — same bucket math at a coarser
// resolution. Kept as a separate table rather than a tier-tagged one for
// clean indexes and zero CASE-WHEN cruft in queries.
type Sample1h struct {
	ent.Schema
}

// Fields lists the schema fields. Required by Ent's schema interface.
func (Sample1h) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("bucket_start_unix_ms").
			Comment("Unix-ms of the 1-hour window start, multiple of 3_600_000"),
		field.String("target_label").
			NotEmpty(),
		field.Int("count"),
		field.Int("success_count"),
		field.Int("fail_count"),
		field.Float("rtt_p50_ms").Optional().Nillable(),
		field.Float("rtt_p95_ms").Optional().Nillable(),
		field.Float("rtt_p99_ms").Optional().Nillable(),
		field.Float("rtt_max_ms").Optional().Nillable(),
		field.Float("rtt_mean_ms").Optional().Nillable(),
		field.Float("jitter_ms").Optional().Nillable(),
		field.JSON("errors", map[string]int{}).Optional(),
	}
}

// Indexes lists the schema indexes. Required by Ent's schema interface.
func (Sample1h) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("bucket_start_unix_ms", "target_label").Unique(),
		index.Fields("bucket_start_unix_ms"),
	}
}
