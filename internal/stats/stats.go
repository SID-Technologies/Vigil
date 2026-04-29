// Package stats provides pure analysis functions used by the aggregator and report generator.
package stats

import (
	"math"
	"sort"

	"github.com/sid-technologies/vigil/internal/constants"
)

// Percentile returns the value at quantile q (0..1) from a sorted slice
// using floor(len*q), clamped. Empty input returns (0, false).
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

// JitterMs is RFC 3550-style jitter: the mean absolute delta between
// consecutive RTTs in time order. This is what voice/video codecs feel;
// std-dev-of-all misses bursty jitter that matters for call quality.
// Caller orders by timestamp ascending; mixing targets gives nonsense.
// Returns (0, false) for fewer than 2 samples.
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

// BucketSummary is what the aggregator writes to sample_5min/sample_1h.
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

// SampleInput is the minimal projection of a probe Sample for aggregation.
type SampleInput struct {
	TSUnixMs int64
	Success  bool
	RTTMs    *float64
	Error    *string
}

// Aggregate folds a slice of samples (any order) into a BucketSummary.
// Stats pointer fields stay nil when there are no successful samples — that
// distinguishes "no signal" from "0ms latency" in the UI.
func Aggregate(samples []SampleInput) BucketSummary {
	out := BucketSummary{
		Count:  len(samples),
		Errors: map[string]int{},
	}

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

// FillBucketRTTStats populates percentile/max/mean/jitter pointer fields
// from RTTs in time order. No-op when empty.
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

// AggregateFromBuckets folds child summaries into a coarser one for the
// 5min→1h rollup. Counts/errors sum exactly. Percentiles and jitter use
// success-count-weighted means of the child values — a fidelity trade-off
// once raw RTTs have been pruned.
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

// Round2 rounds v to two decimal places. Single definition prevents drift
// between probe recording and report generation.
func Round2(v float64) float64 {
	const hundredths = 100.0

	return math.Round(v*hundredths) / hundredths
}
