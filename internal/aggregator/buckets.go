// Package aggregator owns the rollup goroutine that builds 1-min, 5-min,
// and 1-hour aggregated buckets from raw probe samples.
package aggregator

import "github.com/sid-technologies/vigil/internal/constants"

// Bucket widths re-exported from `internal/constants` so callers in this
// package don't need to import constants directly for the common case.
const (
	OneMinMs       = constants.OneMinBucketMs
	FiveMinMs      = constants.FiveMinBucketMs
	OneHourMs      = constants.OneHourBucketMs
	SafetyMarginMs = constants.SafetyMarginMs
)

// FloorBucket returns the start of the bucket containing tsMs at the given
// width.
func FloorBucket(tsMs, widthMs int64) int64 {
	return (tsMs / widthMs) * widthMs
}

// ClosedBucketRange returns the inclusive bucket-start range [oldest, newest]
// that the aggregator should consider closed-and-ready at wall-clock nowMs.
// The newest bucket is `now - safety` floored; the oldest is bounded by
// lookbackMs so we don't scan years of history every wakeup.
//
// Returns (newest < oldest) when there's nothing to consider — caller skips.
func ClosedBucketRange(nowMs, widthMs, lookbackMs int64) (oldest, newest int64) { //nolint:nonamedreturns // revive's confusing-results wants named returns for same-type tuples
	newest = FloorBucket(nowMs-SafetyMarginMs, widthMs)
	oldest = max(newest-lookbackMs, 0)

	return oldest, newest
}
