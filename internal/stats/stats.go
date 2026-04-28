// Package stats provides pure analysis functions used by the aggregator
// and the report generator. No I/O, no logging — easy to unit-test.
package stats

import (
	"sort"

	"github.com/sid-technologies/vigil/internal/constants"
)

// Percentile returns the value at quantile q (0..1) from a sorted slice.
// Uses linear-index style: idx = floor(len * q), clamped to last element.
// Matches the Python tool's behavior so historical reports remain comparable.
//
// Returns (0, false) if the slice is empty.
func Percentile(sortedXs []float64, q float64) (float64, bool) {
	if len(sortedXs) == 0 {
		return 0, false
	}

	idx := int(float64(len(sortedXs)) * q)
	if idx >= len(sortedXs) {
		idx = len(sortedXs) - 1
	}

	if idx < 0 {
		idx = 0
	}

	return sortedXs[idx], true
}

// Mean of an unsorted slice. Returns (0, false) when empty.
func Mean(xs []float64) (float64, bool) {
	if len(xs) == 0 {
		return 0, false
	}

	sum := 0.0
	for _, v := range xs {
		sum += v
	}

	return sum / float64(len(xs)), true
}

// JitterMs is RFC 3550-style jitter — the mean absolute delta between
// consecutive RTTs in time order. This is what voice/video codecs actually
// feel; std-dev-of-all is the wrong metric for call quality and would have
// missed bursty jitter that matters in practice.
//
// Requires at least 2 samples. Returns (0, false) otherwise.
//
// NOTE: caller is responsible for ordering the slice by timestamp ascending
// — jitter math depends on it. Mixing two targets' RTTs into one slice gives
// nonsense.
func JitterMs(rttsInTimeOrder []float64) (float64, bool) {
	if len(rttsInTimeOrder) < 2 {
		return 0, false
	}

	sumAbs := 0.0

	for i := 1; i < len(rttsInTimeOrder); i++ {
		d := rttsInTimeOrder[i] - rttsInTimeOrder[i-1]
		if d < 0 {
			d = -d
		}

		sumAbs += d
	}

	return sumAbs / float64(len(rttsInTimeOrder)-1), true
}

// BucketSummary is the output of Aggregate — what the aggregator writes to
// sample_5min/sample_1h tables (minus the bucket key itself).
type BucketSummary struct {
	Count        int
	SuccessCount int
	FailCount    int
	P50Ms        *float64
	P95Ms        *float64
	P99Ms        *float64
	MaxMs        *float64
	MeanMs       *float64
	JitterMs     *float64
	Errors       map[string]int
}

// SampleInput is the minimal projection of a probe Sample needed to compute
// a BucketSummary. Decoupled from the Ent type so tests don't need a DB.
type SampleInput struct {
	TSUnixMs int64
	Success  bool
	RTTMs    *float64
	Error    *string
}

// Aggregate folds a slice of samples (any time order) into a BucketSummary.
// Sorts internally so the caller can pass DB query results as-is.
//
// All percentiles / mean / max / jitter are computed only over successful
// samples with non-nil RTT. Error counts are computed over failed samples.
//
// Pointer fields (P50Ms, P95Ms, P99Ms, MaxMs, MeanMs, JitterMs) are nil
// when there are no successful samples — distinguishes "no signal" from
// "0ms latency" in the UI.
func Aggregate(samples []SampleInput) BucketSummary {
	out := BucketSummary{
		Count:  len(samples),
		Errors: map[string]int{},
	}

	// Two passes: collect time-ordered RTTs for jitter, sorted RTTs for
	// percentiles, and tally errors.
	timeOrdered := make([]SampleInput, len(samples))
	copy(timeOrdered, samples)
	sort.Slice(timeOrdered, func(i, j int) bool {
		return timeOrdered[i].TSUnixMs < timeOrdered[j].TSUnixMs
	})

	rttsInTimeOrder := make([]float64, 0, len(samples))

	for _, s := range timeOrdered {
		if s.Success {
			out.SuccessCount++

			if s.RTTMs != nil {
				rttsInTimeOrder = append(rttsInTimeOrder, *s.RTTMs)
			}

			continue
		}

		out.FailCount++
		if s.Error != nil {
			out.Errors[*s.Error]++
		} else {
			out.Errors["unknown"]++
		}
	}

	FillBucketRTTStats(&out, rttsInTimeOrder)

	if len(out.Errors) == 0 {
		out.Errors = nil
	}

	return out
}

// FillBucketRTTStats populates the percentile, max, mean, and jitter pointer
// fields on out from a slice of RTTs in time order. No-op when empty.
func FillBucketRTTStats(out *BucketSummary, rttsInTimeOrder []float64) {
	if len(rttsInTimeOrder) == 0 {
		return
	}

	rttsSorted := make([]float64, len(rttsInTimeOrder))
	copy(rttsSorted, rttsInTimeOrder)
	sort.Float64s(rttsSorted)

	if v, ok := Percentile(rttsSorted, constants.P50Quantile); ok {
		vv := Round2(v)
		out.P50Ms = &vv
	}

	if v, ok := Percentile(rttsSorted, constants.P95Quantile); ok {
		vv := Round2(v)
		out.P95Ms = &vv
	}

	if v, ok := Percentile(rttsSorted, constants.P99Quantile); ok {
		vv := Round2(v)
		out.P99Ms = &vv
	}

	maxMs := Round2(rttsSorted[len(rttsSorted)-1])
	out.MaxMs = &maxMs

	if mean, ok := Mean(rttsSorted); ok {
		mean = Round2(mean)
		out.MeanMs = &mean
	}

	if j, ok := JitterMs(rttsInTimeOrder); ok {
		j = Round2(j)
		out.JitterMs = &j
	}
}

// AggregateFromBuckets folds child bucket summaries into a coarser summary.
// Used by the 5min→1h rollup. Counts and error tallies sum directly. Stats
// (percentiles, jitter) are recomputed from rebuilt-but-approximate inputs:
//
//   - We don't have the raw RTTs anymore at the 5min level.
//   - For the 1-hour bucket's percentiles we approximate with weighted
//     averages of the child buckets' p50/p95/p99/max/mean. This is a
//     well-known trade-off — slightly less precise than re-percentiling
//     raw, but raw has been pruned by then.
//   - Jitter is the mean of child-bucket jitter values (weighted by sample
//     count). Loses some signal but stays directionally correct.
//
// For full statistical fidelity we'd need to keep all raw forever —
// trade-off is intentional and documented in CLAUDE.md.
func AggregateFromBuckets(children []BucketSummary) BucketSummary {
	out := BucketSummary{Errors: map[string]int{}}
	if len(children) == 0 {
		return out
	}

	var (
		weightedP50, weightedP95, weightedP99, weightedMean, weightedJitter float64
		p50W, p95W, p99W, meanW, jitterW                                    int
		maxSoFar                                                            *float64
	)

	for _, c := range children {
		out.Count += c.Count
		out.SuccessCount += c.SuccessCount
		out.FailCount += c.FailCount

		for k, v := range c.Errors {
			out.Errors[k] += v
		}

		if c.P50Ms != nil {
			weightedP50 += *c.P50Ms * float64(c.SuccessCount)
			p50W += c.SuccessCount
		}

		if c.P95Ms != nil {
			weightedP95 += *c.P95Ms * float64(c.SuccessCount)
			p95W += c.SuccessCount
		}

		if c.P99Ms != nil {
			weightedP99 += *c.P99Ms * float64(c.SuccessCount)
			p99W += c.SuccessCount
		}

		if c.MeanMs != nil {
			weightedMean += *c.MeanMs * float64(c.SuccessCount)
			meanW += c.SuccessCount
		}

		if c.JitterMs != nil {
			weightedJitter += *c.JitterMs * float64(c.SuccessCount)
			jitterW += c.SuccessCount
		}

		if c.MaxMs != nil {
			if maxSoFar == nil || *c.MaxMs > *maxSoFar {
				v := *c.MaxMs
				maxSoFar = &v
			}
		}
	}

	if p50W > 0 {
		v := Round2(weightedP50 / float64(p50W))
		out.P50Ms = &v
	}

	if p95W > 0 {
		v := Round2(weightedP95 / float64(p95W))
		out.P95Ms = &v
	}

	if p99W > 0 {
		v := Round2(weightedP99 / float64(p99W))
		out.P99Ms = &v
	}

	if meanW > 0 {
		v := Round2(weightedMean / float64(meanW))
		out.MeanMs = &v
	}

	if jitterW > 0 {
		v := Round2(weightedJitter / float64(jitterW))
		out.JitterMs = &v
	}

	out.MaxMs = maxSoFar

	if len(out.Errors) == 0 {
		out.Errors = nil
	}

	return out
}

// Round2 rounds v to two decimal places. Exported because probe recording
// and report generation use the same rounding — single definition prevents
// drift across call sites.
func Round2(v float64) float64 {
	const (
		hundredths = 100.0
		halfStep   = 0.5
	)

	return float64(int64(v*hundredths+halfStep)) / hundredths
}
