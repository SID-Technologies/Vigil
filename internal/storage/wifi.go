package storage

import (
	"context"

	"github.com/sid-technologies/vigil/db/ent"
	"github.com/sid-technologies/vigil/db/ent/wifisample"
)

// WifiSample is the IPC-shape view of one Wi-Fi observation.
type WifiSample struct {
	TSUnixMs      int64    `json:"ts_unix_ms"`
	SSID          *string  `json:"ssid,omitempty"`
	BSSID         *string  `json:"bssid,omitempty"`
	SignalPercent *int     `json:"signal_percent,omitempty"`
	RSSIDbm       *int     `json:"rssi_dbm,omitempty"`
	RxRateMbps    *float64 `json:"rx_rate_mbps,omitempty"`
	TxRateMbps    *float64 `json:"tx_rate_mbps,omitempty"`
	Channel       *string  `json:"channel,omitempty"`
}

// QueryWifiSamples returns Wi-Fi samples in [fromMs, toMs] ordered by time.
// One sample/minute means a 7-day window is ~10K rows — no pagination needed.
func (s *Store) QueryWifiSamples(ctx context.Context, fromMs, toMs int64) ([]WifiSample, error) {
	rows, err := s.client.WifiSample.Query().
		Where(
			wifisample.TsUnixMsGTE(fromMs),
			wifisample.TsUnixMsLTE(toMs),
		).
		Order(ent.Asc(wifisample.FieldTsUnixMs)).
		All(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // wrapped at IPC boundary
	}

	out := make([]WifiSample, 0, len(rows))
	for _, r := range rows {
		w := WifiSample{TSUnixMs: r.TsUnixMs}
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
