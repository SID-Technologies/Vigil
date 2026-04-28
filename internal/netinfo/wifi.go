package netinfo

import "time"

// WifiSample captures network attachment state at a point in time. Most
// fields are platform-dependent and may be nil; SignalPercent and Rx/Tx
// rates are Windows-only.
type WifiSample struct {
	Timestamp     time.Time
	SSID          *string
	BSSID         *string
	SignalPercent *int     // Windows-only
	RSSIDbm       *int     // best-effort cross-platform
	RxRateMbps    *float64 // Windows-only
	TxRateMbps    *float64 // Windows-only
	Channel       *string
}

func strPtr(s string) *string { return &s }

func intPtr(i int) *int { return &i }

//nolint:unused // used by platform-specific wifi_*.go files via build tags
func f64Ptr(f float64) *float64 { return &f }
