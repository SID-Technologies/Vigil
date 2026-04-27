package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Outage records a detected reachability gap. Detected live by the outage
// detector goroutine (3+ consecutive failures of the same scope) and never
// re-aggregated — outages are too valuable to lose to retention pruning,
// and small enough that keeping them forever costs nothing.
//
// Two scope kinds:
//
//   - "target:<label>" — that one target failed N consecutive cycles.
//   - "network"        — every probe failed in N consecutive cycles. The
//                        connection was actually down, not just one
//                        endpoint having a bad day.
//
// `end_ts_unix_ms` is null while the outage is ongoing. The detector sets
// it on the first successful cycle that follows the failure run. UI uses
// `end_ts == null` to show a live "outage in progress" badge.
type Outage struct {
	ent.Schema
}

func (Outage) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			Unique().
			Immutable().
			Comment("UUID set at creation"),
		field.String("scope").
			NotEmpty().
			Comment("'network' or 'target:<label>'"),
		field.Int64("start_ts_unix_ms"),
		field.Int64("end_ts_unix_ms").
			Optional().
			Nillable().
			Comment("Null while ongoing"),
		field.Int("consecutive_failures"),
		field.JSON("errors", map[string]int{}).Optional().
			Comment("Error code → count across the failure run"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}

func (Outage) Indexes() []ent.Index {
	return []ent.Index{
		// "Show me outages from the last 7d, optionally filtered by scope."
		index.Fields("scope", "start_ts_unix_ms"),
		// "Find the open outage for this scope" — used by the detector when
		// a success arrives to close the active outage row.
		index.Fields("scope", "end_ts_unix_ms"),
	}
}
