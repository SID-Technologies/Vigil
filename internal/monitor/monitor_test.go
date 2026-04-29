//nolint:testpackage // whitebox test for unexported computeCycleTimeout
package monitor

import (
	"testing"
	"time"
)

func TestComputeCycleTimeout(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		probeMs   int
		wantLower time.Duration
		wantUpper time.Duration
	}{
		// 1000ms probe → 1500ms base + 500ms slack = 2s
		{"typical", 1000, 1900 * time.Millisecond, 2100 * time.Millisecond},
		// 2500ms probe → 3750ms + 500ms = 4250ms
		{"long_probe", 2500, 4200 * time.Millisecond, 4300 * time.Millisecond},
		// Even a zero probe gets the 500ms scheduler margin.
		{"zero_probe_keeps_scheduler_margin", 0, 500 * time.Millisecond, 500 * time.Millisecond},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := computeCycleTimeout(tc.probeMs)
			if got < tc.wantLower || got > tc.wantUpper {
				t.Fatalf("computeCycleTimeout(%d) = %v, want %v..%v",
					tc.probeMs, got, tc.wantLower, tc.wantUpper)
			}
		})
	}
}
