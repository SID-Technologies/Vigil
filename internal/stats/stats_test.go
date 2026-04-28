package stats_test

import (
	"math"
	"testing"
	"github.com/sid-technologies/vigil/internal/stats"
)

func TestPercentile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  []float64
		q      float64
		wantOK bool
		want   float64
	}{
		{name: "empty", input: nil, q: 0.5, wantOK: false},
		{name: "single", input: []float64{42}, q: 0.5, wantOK: true, want: 42},
		{name: "p50 of ten", input: []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, q: 0.5, wantOK: true, want: 6},
		{name: "p95 of ten", input: []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, q: 0.95, wantOK: true, want: 10},
		{name: "p99 of one hundred", input: hundredAscending(), q: 0.99, wantOK: true, want: 100},
		{name: "q clamped above 1", input: []float64{1, 2, 3}, q: 1.5, wantOK: true, want: 3},
		{name: "q at zero", input: []float64{1, 2, 3}, q: 0, wantOK: true, want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := stats.Percentile(tt.input, tt.q)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}

			if ok && got != tt.want {
				t.Errorf("stats.Percentile(%v, %v) = %v, want %v", tt.input, tt.q, got, tt.want)
			}
		})
	}
}

func TestMean(t *testing.T) {
	t.Parallel()

	if _, ok := stats.Mean(nil); ok {
		t.Error("stats.Mean(nil) returned ok, want !ok")
	}

	tests := []struct {
		name  string
		input []float64
		want  float64
	}{
		{name: "single", input: []float64{42}, want: 42},
		{name: "two", input: []float64{1, 3}, want: 2},
		{name: "spread", input: []float64{1, 2, 3, 4, 5}, want: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := stats.Mean(tt.input)
			if !ok {
				t.Fatal("Mean returned !ok")
			}

			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("stats.Mean(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestJitterMs(t *testing.T) {
	t.Parallel()

	t.Run("requires at least two", func(t *testing.T) {
		t.Parallel()

		if _, ok := stats.JitterMs(nil); ok {
			t.Error("stats.JitterMs(nil) returned ok")
		}

		if _, ok := stats.JitterMs([]float64{42}); ok {
			t.Error("stats.JitterMs(single) returned ok")
		}
	})

	t.Run("constant series has zero jitter", func(t *testing.T) {
		t.Parallel()

		j, ok := stats.JitterMs([]float64{10, 10, 10, 10})
		if !ok {
			t.Fatal("ok = false")
		}

		if j != 0 {
			t.Errorf("stats.JitterMs(constant) = %v, want 0", j)
		}
	})

	t.Run("alternating series matches absolute deltas", func(t *testing.T) {
		t.Parallel()
		// Deltas: |20-10|=10, |10-20|=10, |20-10|=10 → mean = 10
		j, ok := stats.JitterMs([]float64{10, 20, 10, 20})
		if !ok {
			t.Fatal("ok = false")
		}

		if j != 10 {
			t.Errorf("stats.JitterMs(alt) = %v, want 10", j)
		}
	})

	t.Run("monotonic increasing", func(t *testing.T) {
		t.Parallel()
		// Deltas: 1,1,1,1 → mean 1
		j, ok := stats.JitterMs([]float64{10, 11, 12, 13, 14})
		if !ok {
			t.Fatal("ok = false")
		}

		if math.Abs(j-1) > 1e-9 {
			t.Errorf("stats.JitterMs(monotonic) = %v, want 1", j)
		}
	})
}

func TestAggregate(t *testing.T) {
	t.Parallel()

	t.Run("empty produces zero count", func(t *testing.T) {
		t.Parallel()

		out := stats.Aggregate(nil)
		if out.Count != 0 || out.SuccessCount != 0 || out.FailCount != 0 {
			t.Errorf("stats.Aggregate(nil) = %+v", out)
		}

		if out.P50Ms != nil || out.P95Ms != nil {
			t.Error("stats.Aggregate(nil) had non-nil percentiles")
		}
	})

	t.Run("all successes computes percentiles", func(t *testing.T) {
		t.Parallel()

		samples := []stats.SampleInput{
			{TSUnixMs: 1, Success: true, RTTMs: ptr(10.0)},
			{TSUnixMs: 2, Success: true, RTTMs: ptr(20.0)},
			{TSUnixMs: 3, Success: true, RTTMs: ptr(30.0)},
			{TSUnixMs: 4, Success: true, RTTMs: ptr(40.0)},
			{TSUnixMs: 5, Success: true, RTTMs: ptr(50.0)},
		}

		out := stats.Aggregate(samples)
		if out.Count != 5 || out.SuccessCount != 5 || out.FailCount != 0 {
			t.Errorf("counts wrong: %+v", out)
		}

		if out.P50Ms == nil || *out.P50Ms != 30 {
			t.Errorf("p50 = %v, want 30", out.P50Ms)
		}

		if out.MaxMs == nil || *out.MaxMs != 50 {
			t.Errorf("max = %v, want 50", out.MaxMs)
		}

		if out.MeanMs == nil || *out.MeanMs != 30 {
			t.Errorf("mean = %v, want 30", out.MeanMs)
		}
	})

	t.Run("mixed success and failure tallies errors", func(t *testing.T) {
		t.Parallel()

		samples := []stats.SampleInput{
			{TSUnixMs: 1, Success: true, RTTMs: ptr(10.0)},
			{TSUnixMs: 2, Success: false, Error: ptr("timeout")},
			{TSUnixMs: 3, Success: false, Error: ptr("timeout")},
			{TSUnixMs: 4, Success: false, Error: ptr("dns")},
			{TSUnixMs: 5, Success: true, RTTMs: ptr(20.0)},
		}

		out := stats.Aggregate(samples)
		if out.SuccessCount != 2 || out.FailCount != 3 {
			t.Errorf("counts wrong: %+v", out)
		}

		if out.Errors["timeout"] != 2 || out.Errors["dns"] != 1 {
			t.Errorf("errors wrong: %v", out.Errors)
		}
	})

	t.Run("all failures has no percentiles", func(t *testing.T) {
		t.Parallel()

		samples := []stats.SampleInput{
			{TSUnixMs: 1, Success: false, Error: ptr("timeout")},
			{TSUnixMs: 2, Success: false, Error: ptr("timeout")},
		}

		out := stats.Aggregate(samples)
		if out.P50Ms != nil || out.MaxMs != nil {
			t.Errorf("expected nil percentiles, got %+v", out)
		}
	})

	t.Run("nil error code falls back to unknown", func(t *testing.T) {
		t.Parallel()

		samples := []stats.SampleInput{{TSUnixMs: 1, Success: false}}

		out := stats.Aggregate(samples)
		if out.Errors["unknown"] != 1 {
			t.Errorf("expected unknown=1, got %v", out.Errors)
		}
	})
}

func TestAggregateFromBuckets(t *testing.T) {
	t.Parallel()

	t.Run("empty input is empty summary", func(t *testing.T) {
		t.Parallel()

		out := stats.AggregateFromBuckets(nil)
		if out.Count != 0 {
			t.Errorf("expected zero count, got %d", out.Count)
		}
	})

	t.Run("counts and errors sum across buckets", func(t *testing.T) {
		t.Parallel()

		children := []stats.BucketSummary{
			{
				Count: 100, SuccessCount: 95, FailCount: 5,
				P50Ms: ptr(20.0), P95Ms: ptr(40.0), P99Ms: ptr(50.0), MaxMs: ptr(60.0), MeanMs: ptr(22.0),
				Errors: map[string]int{"timeout": 3, "dns": 2},
			},
			{
				Count: 100, SuccessCount: 100, FailCount: 0,
				P50Ms: ptr(10.0), P95Ms: ptr(15.0), P99Ms: ptr(18.0), MaxMs: ptr(20.0), MeanMs: ptr(11.0),
			},
		}

		out := stats.AggregateFromBuckets(children)
		if out.Count != 200 {
			t.Errorf("count = %d, want 200", out.Count)
		}

		if out.SuccessCount != 195 || out.FailCount != 5 {
			t.Errorf("success/fail = %d/%d, want 195/5", out.SuccessCount, out.FailCount)
		}
		// max should be the highest of the children — 60
		if out.MaxMs == nil || *out.MaxMs != 60 {
			t.Errorf("max = %v, want 60", out.MaxMs)
		}
		// p50 weighted mean: (20*95 + 10*100) / 195 ≈ 14.87
		if out.P50Ms == nil || math.Abs(*out.P50Ms-14.87) > 0.01 {
			t.Errorf("p50 = %v, want ≈14.87", out.P50Ms)
		}

		if out.Errors["timeout"] != 3 {
			t.Errorf("error tally lost — got %v", out.Errors)
		}
	})
}

// ============================================================================
// Helpers
// ============================================================================

func ptr[T any](v T) *T { return &v }

func hundredAscending() []float64 {
	out := make([]float64, 100)
	for i := range out {
		out[i] = float64(i + 1)
	}

	return out
}
