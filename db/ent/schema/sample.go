package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Sample is a single raw probe result. High write volume (every 2.5s × 13
// targets ≈ 5/s steady-state), so we use Ent's default int auto-increment ID
// rather than UUIDs — UUIDs are wasteful for high-frequency time-series.
//
// This is a deliberate departure from Pugio's UUID-everywhere convention,
// documented here. Aggregated tables (Sample5min, Sample1h) follow the same
// pattern when added in phase 3.
//
// Retention: pruned after retention_raw_days from app_config (default 7 days).
//
// We denormalize target_label/kind/host/port at insert time instead of using
// an edge to Target. Reasons:
//   1. Targets can be renamed or deleted; samples must retain their original
//      identity for historical reports ("show me May 2026 outages on the old
//      router_icmp target").
//   2. Avoids a join on the hottest query path (per-target time-windowed reads).
//   3. Keeps each row self-contained for CSV export.
type Sample struct {
	ent.Schema
}

// Fields lists the schema fields. Required by Ent's schema interface.
func (Sample) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("ts_unix_ms").
			Comment("Probe timestamp, milliseconds since unix epoch"),
		field.String("target_label").
			NotEmpty(),
		field.String("target_kind"),
		field.String("target_host"),
		field.Int("target_port").
			Optional().
			Nillable(),
		field.Bool("success"),
		field.Float("rtt_ms").
			Optional().
			Nillable().
			Comment("Round-trip time, only set when success=true"),
		field.String("error").
			Optional().
			Nillable().
			Comment("Stable machine-readable error code: timeout, host_unreachable, dns, conn_refused, etc."),
	}
}

// Indexes lists the schema indexes. Required by Ent's schema interface.
func (Sample) Indexes() []ent.Index {
	return []ent.Index{
		// Time-range scans for the History page.
		index.Fields("ts_unix_ms"),
		// Per-target time-windowed queries (the dashboard's hot path).
		index.Fields("target_label", "ts_unix_ms"),
	}
}
