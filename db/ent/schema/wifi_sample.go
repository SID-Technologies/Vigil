package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// WifiSample captures Wi-Fi state at flush time. Most fields are nullable
// because availability varies wildly by platform:
//
//   - signal_percent and rx/tx rates: Windows-only (parsed from `netsh wlan
//     show interfaces`). Other platforms expose nothing equivalent without
//     elevated permissions.
//   - rssi_dbm: macOS via `system_profiler SPAirPortDataType`, Windows
//     derived from signal_percent, Linux via netlink.
//   - ssid/bssid/channel: usually available on all platforms, sometimes
//     redacted on macOS without Location Services permission.
//
// Frontend treats nulls as "n/a", not zero.
type WifiSample struct {
	ent.Schema
}

// Fields lists the schema fields. Required by Ent's schema interface.
func (WifiSample) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("ts_unix_ms"),
		field.String("ssid").
			Optional().
			Nillable(),
		field.String("bssid").
			Optional().
			Nillable(),
		field.Int("signal_percent").
			Optional().
			Nillable().
			Comment("Windows-only — derived elsewhere"),
		field.Int("rssi_dbm").
			Optional().
			Nillable(),
		field.Float("rx_rate_mbps").
			Optional().
			Nillable().
			Comment("Windows-only"),
		field.Float("tx_rate_mbps").
			Optional().
			Nillable().
			Comment("Windows-only"),
		field.String("channel").
			Optional().
			Nillable(),
	}
}

// Indexes lists the schema indexes. Required by Ent's schema interface.
func (WifiSample) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("ts_unix_ms"),
	}
}
