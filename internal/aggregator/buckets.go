// Package aggregator builds 1-min, 5-min, and 1-hour rollup buckets from raw probe samples.
package aggregator

import "github.com/sid-technologies/vigil/internal/constants"

// Re-exports of constants used throughout this package.
const (
	OneMinMs       = constants.OneMinBucketMs
	FiveMinMs      = constants.FiveMinBucketMs
	OneHourMs      = constants.OneHourBucketMs
	SafetyMarginMs = constants.SafetyMarginMs
)

// FloorBucket returns the start of the bucket containing tsMs.
func FloorBucket(tsMs, widthMs int64) int64 {
	return (tsMs / widthMs) * widthMs
}

// ClosedBucketRange returns the inclusive [oldest, newest] bucket-start range
// the aggregator should consider closed-and-ready at nowMs. Returns
// newest < oldest when nothing is ready.
func ClosedBucketRange(nowMs, widthMs, lookbackMs int64) (oldest, newest int64) { //nolint:nonamedreturns // revive's confusing-results wants named returns for same-type tuples
	newest = FloorBucket(nowMs-SafetyMarginMs, widthMs)
	oldest = max(newest-lookbackMs, 0)

	return oldest, newest
}
