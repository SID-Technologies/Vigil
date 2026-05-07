// Package stats provides pure analysis functions used by the aggregator and report generator.
package stats

import (
	"math"
	"sort"

	"github.com/sid-technologies/vigil/internal/constants"
)

// SampleInput is the minimal projection of a probe Sample for aggregation.
type SampleInput struct {
	TSUnixMs int64
	Success  bool
	RTTMs    *float64
	Error    *string
}

// BucketSummary is the row written to sample_5min / sample_1h. Pointer fields
// stay nil when there are no successful samples — distinguishes "no signal"
// from "0ms latency" in the UI.
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

type weightedAvg struct {
	sum    float64
	weight int
}

func (a *weightedAvg) add(value *float64, weight int) {
	if value == nil {
		return
	}

	a.sum += *value * float64(weight)
	a.weight += weight
}

func (a *weightedAvg) result() *float64 {
	if a.weight == 0 {
		return nil
	}

	v := Round2(a.sum / float64(a.weight))

	return &v
}

// Round2 rounds v to two decimal places. Single definition prevents drift
// between probe recording and report generation.
func Round2(v float64) float64 {
	const hundredths = 100.0

	return math.Round(v*hundredths) / hundredths
}

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

// roundedPercentile is a small wrapper around Percentile that rounds and
// boxes the result so callers can assign directly to a *float64 field.
func roundedPercentile(sortedXs []float64, q float64) *float64 {
	v, ok := Percentile(sortedXs, q)
	if !ok {
		return nil
	}

	r := Round2(v)

	return &r
}

// Aggregate folds a slice of samples (any order) into a BucketSummary.
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

		errKey := "unknown"
		if s.Error != nil {
			errKey = *s.Error
		}

		out.Errors[errKey]++
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

	out.P50Ms = roundedPercentile(rttsSorted, constants.P50Quantile)
	out.P95Ms = roundedPercentile(rttsSorted, constants.P95Quantile)
	out.P99Ms = roundedPercentile(rttsSorted, constants.P99Quantile)

	maxMs := Round2(rttsSorted[len(rttsSorted)-1])
	out.MaxMs = &maxMs

	if mean, ok := Mean(rttsSorted); ok {
		m := Round2(mean)
		out.MeanMs = &m
	}

	if j, ok := JitterMs(rttsInTimeOrder); ok {
		rj := Round2(j)
		out.JitterMs = &rj
	}
}

// AggregateFromBuckets folds child summaries into a coarser one for the
// 5min → 1h rollup. Counts/errors sum exactly. Percentiles and jitter use
// success-count-weighted means of the child values — see weightedAvg.
func AggregateFromBuckets(children []BucketSummary) BucketSummary {
	out := BucketSummary{Errors: map[string]int{}}
	if len(children) == 0 {
		return out
	}

	var p50, p95, p99, mean, jitter weightedAvg

	var maxSoFar *float64

	for _, c := range children {
		out.Count += c.Count
		out.SuccessCount += c.SuccessCount
		out.FailCount += c.FailCount

		for k, v := range c.Errors {
			out.Errors[k] += v
		}

		p50.add(c.P50Ms, c.SuccessCount)
		p95.add(c.P95Ms, c.SuccessCount)
		p99.add(c.P99Ms, c.SuccessCount)
		mean.add(c.MeanMs, c.SuccessCount)
		jitter.add(c.JitterMs, c.SuccessCount)

		if c.MaxMs != nil && (maxSoFar == nil || *c.MaxMs > *maxSoFar) {
			v := *c.MaxMs
			maxSoFar = &v
		}
	}

	out.P50Ms = p50.result()
	out.P95Ms = p95.result()
	out.P99Ms = p99.result()
	out.MeanMs = mean.result()
	out.JitterMs = jitter.result()
	out.MaxMs = maxSoFar

	if len(out.Errors) == 0 {
		out.Errors = nil
	}

	return out
}
