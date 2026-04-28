// Package constants holds time math, config defaults, and statistical thresholds.
// Anything ending in `Ms` is unix-millisecond integer math (the wire format).
package constants

// Time conversions in unix milliseconds.
const (
	OneSecondMs int64 = 1_000
	OneMinuteMs int64 = 60 * OneSecondMs
	OneHourMs   int64 = 60 * OneMinuteMs
	OneDayMs    int64 = 24 * OneHourMs
	OneWeekMs   int64 = 7 * OneDayMs
)

// Aggregation bucket widths. Changing these invalidates existing buckets.
const (
	OneMinBucketMs  int64 = OneMinuteMs
	FiveMinBucketMs int64 = 5 * OneMinuteMs
	OneHourBucketMs int64 = OneHourMs
)

// Per-tier lookback windows on each aggregator wakeup.
const (
	DefaultLookback1MinMs int64 = 6 * OneHourMs
	DefaultLookback5MinMs int64 = OneDayMs
	DefaultLookback1HMs   int64 = OneWeekMs
)

// SafetyMarginMs delays bucket close past the flush interval (60s) so all samples are durable.
const SafetyMarginMs int64 = 90 * OneSecondMs

// app_config seed defaults.
const (
	DefaultPingIntervalSec  float64 = 2.5
	DefaultPingTimeoutMs    int     = 2000
	DefaultFlushIntervalSec int     = 60

	DefaultRetentionRawDays  int = 7
	DefaultRetention1MinDays int = 14
	DefaultRetention5MinDays int = 90
)

// Latency quantiles and the 0..1 → 0..100 conversion factor.
const (
	P50Quantile float64 = 0.5
	P95Quantile float64 = 0.95
	P99Quantile float64 = 0.99

	PercentMultiplier float64 = 100.0
)
