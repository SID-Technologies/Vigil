//nolint:testpackage // whitebox test for unexported pickGranularity
package ipc

import (
	"testing"
)

func TestPickGranularity(t *testing.T) {
	t.Parallel()

	const (
		minute = int64(60 * 1000)
		hour   = 60 * minute
		day    = 24 * hour
	)

	cases := []struct {
		name     string
		windowMs int64
		want     string
	}{
		{"sub_hour", 30 * minute, granularityRaw},
		{"exactly_hour", hour, granularityRaw},
		{"three_hours", 3 * hour, granularity1Min},
		{"six_hours", 6 * hour, granularity1Min},
		{"twelve_hours", 12 * hour, granularity5min},
		{"seven_days", 7 * day, granularity5min},
		{"thirty_days", 30 * day, granularity1Hour},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := pickGranularity(tc.windowMs)
			if got != tc.want {
				t.Fatalf("pickGranularity(%d) = %q, want %q", tc.windowMs, got, tc.want)
			}
		})
	}
}
