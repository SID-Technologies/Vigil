// Package aggregator owns the rollup goroutine that builds 5-minute and
// 1-hour aggregated buckets from raw probe samples.
package aggregator

const (
	// FiveMinMs is the bucket width for sample_5min, in unix-ms. Hardcoded
	// because changing it would invalidate all existing buckets.
	FiveMinMs int64 = 5 * 60 * 1000

	// OneHourMs is the bucket width for sample_1h.
	OneHourMs int64 = 60 * 60 * 1000

	// SafetyMarginMs is how long after a bucket's end we wait before
	// considering it "closed" and writing its aggregation. Guards against
	// late-arriving samples (e.g. a probe that took longer than expected
	// and crossed the bucket boundary). Slightly larger than the default
	// flush interval (60s) so all flushed-but-not-yet-aggregated samples
	// are durable before we read them.
	SafetyMarginMs int64 = 90 * 1000
)

// FloorBucket returns the start of the bucket containing tsMs at the given
// width. Used to compute (target, bucket) keys for upserts.
func FloorBucket(tsMs, widthMs int64) int64 {
	return (tsMs / widthMs) * widthMs
}

// ClosedBucketRange returns the inclusive bucket-start range [oldest, newest]
// that the aggregator should consider "closed and ready" at the given
// wall-clock nowMs. The newest bucket is `now - safety` floored; the oldest
// is bounded by lookbackMs so we don't scan years of history every wakeup.
//
// Returns (newest < oldest) when there's nothing to consider — caller should
// skip.
func ClosedBucketRange(nowMs, widthMs, lookbackMs int64) (oldest, newest int64) {
	newest = FloorBucket(nowMs-SafetyMarginMs, widthMs)
	oldest = newest - lookbackMs
	if oldest < 0 {
		oldest = 0
	}
	return
}
