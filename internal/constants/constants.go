// Package constants centralizes the time-math, default config values, and
// statistical thresholds used across Vigil so call sites read in domain
// terms ("OneDayMs", "DefaultPingIntervalSec") instead of raw arithmetic.
//
// Convention: anything ending in `Ms` is unix-millisecond integer math —
// the wire format Vigil uses everywhere it needs to interop with the
// frontend or persist to SQLite.
package constants

// Time conversions, expressed in unix milliseconds.
const (
	OneSecondMs int64 = 1_000
	OneMinuteMs int64 = 60 * OneSecondMs
	OneHourMs   int64 = 60 * OneMinuteMs
	OneDayMs    int64 = 24 * OneHourMs
	OneWeekMs   int64 = 7 * OneDayMs
)

// Aggregation bucket widths. Hardcoded — changing any of these would
// invalidate every existing bucket in users' databases.
const (
	OneMinBucketMs  int64 = OneMinuteMs
	FiveMinBucketMs int64 = 5 * OneMinuteMs
	OneHourBucketMs int64 = OneHourMs
)

// Aggregator wakeup defaults — how far back each tier scans on every
// cycle. Tuned so a sidecar that's been off for a few hours catches up on
// the first wakeup, but doesn't read months of history every cycle.
const (
	DefaultLookback1MinMs int64 = 6 * OneHourMs
	DefaultLookback5MinMs int64 = OneDayMs
	DefaultLookback1HMs   int64 = OneWeekMs
)

// SafetyMarginMs is how long after a bucket's end the aggregator waits
// before considering it closed. Slightly larger than the default flush
// interval (60s) so all flushed-but-not-yet-aggregated samples are
// durable before the aggregator reads them.
const SafetyMarginMs int64 = 90 * OneSecondMs

// app_config seed defaults. Match the legacy CLI defaults so behavior is
// identical out of the box.
const (
	DefaultPingIntervalSec  float64 = 2.5
	DefaultPingTimeoutMs    int     = 2000
	DefaultFlushIntervalSec int     = 60

	DefaultRetentionRawDays  int = 7
	DefaultRetention1MinDays int = 14
	DefaultRetention5MinDays int = 90
)

// Statistical thresholds and conversion factors used by aggregator and
// reports. P50/P95/P99 are the standard latency-distribution quantiles.
// PercentMultiplier turns a 0..1 ratio into a 0..100 percentage.
const (
	P50Quantile float64 = 0.5
	P95Quantile float64 = 0.95
	P99Quantile float64 = 0.99

	PercentMultiplier float64 = 100.0
)
