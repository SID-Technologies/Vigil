package storage

import (
	"context"

	"github.com/sid-technologies/vigil/db/ent"
	"github.com/sid-technologies/vigil/db/ent/wifisample"
)

// WifiSample is the storage-layer view of a Wi-Fi sample.
type WifiSample struct {
	TsUnixMs      int64    `json:"ts_unix_ms"`
	SSID          *string  `json:"ssid,omitempty"`
	BSSID         *string  `json:"bssid,omitempty"`
	SignalPercent *int     `json:"signal_percent,omitempty"`
	RSSIDbm       *int     `json:"rssi_dbm,omitempty"`
	RxRateMbps    *float64 `json:"rx_rate_mbps,omitempty"`
	TxRateMbps    *float64 `json:"tx_rate_mbps,omitempty"`
	Channel       *string  `json:"channel,omitempty"`
}

// QueryWifiSamples returns Wi-Fi samples in [fromMs, toMs] ordered by time.
// Vigil flushes one Wi-Fi sample per minute by default, so even a 7-day
// window is ~10K rows — no pagination needed at this resolution.
func QueryWifiSamples(ctx context.Context, client *ent.Client, fromMs, toMs int64) ([]WifiSample, error) {
	rows, err := client.WifiSample.Query().
		Where(
			wifisample.TsUnixMsGTE(fromMs),
			wifisample.TsUnixMsLTE(toMs),
		).
		Order(ent.Asc(wifisample.FieldTsUnixMs)).
		All(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	out := make([]WifiSample, 0, len(rows))
	for _, r := range rows {
		w := WifiSample{TsUnixMs: r.TsUnixMs}
		if r.Ssid != nil {
			w.SSID = r.Ssid
		}
		if r.Bssid != nil {
			w.BSSID = r.Bssid
		}
		if r.SignalPercent != nil {
			w.SignalPercent = r.SignalPercent
		}
		if r.RssiDbm != nil {
			w.RSSIDbm = r.RssiDbm
		}
		if r.RxRateMbps != nil {
			w.RxRateMbps = r.RxRateMbps
		}
		if r.TxRateMbps != nil {
			w.TxRateMbps = r.TxRateMbps
		}
		if r.Channel != nil {
			w.Channel = r.Channel
		}
		out = append(out, w)
	}
	return out, nil
}
