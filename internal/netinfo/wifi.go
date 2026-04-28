package netinfo

import "time"

// WifiSample captures the network attachment state at a point in time. Most
// fields are platform-dependent and may be nil:
//
//   - SSID, BSSID, Channel: usually available everywhere.
//   - SignalPercent, RxRateMbps, TxRateMbps: Windows-only (parsed from
//     `netsh wlan show interfaces`).
//   - RSSIDbm: macOS via `system_profiler`, Windows derived, Linux netlink.
//
// Implementations for SampleWifi() live in wifi_{darwin,linux,windows}.go,
// selected at build time via `//go:build` constraints. wifi_other.go is the
// fallback for unsupported platforms (returns the timestamp + nothing else).
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

// strPtr / intPtr / f64Ptr — convenience helpers for the per-platform
// implementations to avoid `x := "foo"; sample.SSID = &x` boilerplate.
//

func strPtr(s string) *string { return &s }

func intPtr(i int) *int { return &i }

//nolint:unused // used by platform-specific wifi_*.go files via build tags
func f64Ptr(f float64) *float64 { return &f }
