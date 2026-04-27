package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// AppConfig is a singleton settings row. There is exactly one row in this
// table at any time, with id=1 (enforced by app code, not the schema). The
// row is created on first run by the seed routine.
//
// Reads happen often (probe loop, flusher) so this is consciously a row
// rather than a JSON file — Ent does in-memory caching above the SQLite
// driver and the row is tiny.
type AppConfig struct {
	ent.Schema
}

func (AppConfig) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").
			Unique().
			Immutable().
			Comment("Always 1 — singleton pattern enforced by application code"),
		field.Float("ping_interval_sec").
			Default(2.5).
			Comment("Seconds between probe cycles"),
		field.Int("flush_interval_sec").
			Default(60).
			Comment("Seconds between disk flushes from in-memory buffer"),
		field.Int("ping_timeout_ms").
			Default(2000).
			Comment("Per-probe timeout in milliseconds"),
		field.Int("retention_raw_days").
			Default(7).
			Comment("Days to retain raw samples before pruning (phase 3)"),
		field.Int("retention_5min_days").
			Default(90).
			Comment("Days to retain 5-minute aggregations (phase 3)"),
		field.Bool("wifi_sample_enabled").
			Default(true),
	}
}
