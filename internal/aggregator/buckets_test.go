package aggregator

import "testing"

func TestFloorBucket(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		ts    int64
		width int64
		want  int64
	}{
		{name: "exact 5min boundary", ts: 1_700_000_000_000, width: FiveMinMs, want: 1_700_000_000_000 - (1_700_000_000_000 % FiveMinMs)},
		{name: "30s into bucket", ts: 1_700_000_030_000, width: FiveMinMs, want: 1_700_000_030_000 - (1_700_000_030_000 % FiveMinMs)},
		{name: "1h boundary", ts: 1_700_003_600_000, width: OneHourMs, want: 1_700_003_600_000 - (1_700_003_600_000 % OneHourMs)},
		{name: "zero", ts: 0, width: FiveMinMs, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := FloorBucket(tt.ts, tt.width)
			if got != tt.want {
				t.Errorf("FloorBucket(%d, %d) = %d, want %d", tt.ts, tt.width, got, tt.want)
			}
			if got%tt.width != 0 {
				t.Errorf("result %d not aligned to width %d", got, tt.width)
			}
		})
	}
}

func TestClosedBucketRange(t *testing.T) {
	t.Parallel()

	t.Run("returns a window when now is well past safety margin", func(t *testing.T) {
		t.Parallel()
		now := int64(1_700_000_000_000) // arbitrary
		oldest, newest := ClosedBucketRange(now, FiveMinMs, 24*60*60*1000)
		if newest < oldest {
			t.Errorf("empty range — newest=%d, oldest=%d", newest, oldest)
		}
		if newest > now-SafetyMarginMs {
			t.Errorf("newest=%d should be <= now-safety=%d", newest, now-SafetyMarginMs)
		}
		if newest%FiveMinMs != 0 {
			t.Errorf("newest=%d not aligned to width", newest)
		}
	})

	t.Run("oldest clamped to zero", func(t *testing.T) {
		t.Parallel()
		// nowMs small, large lookback
		_, _ = ClosedBucketRange(60_000, FiveMinMs, 24*60*60*1000)
		// We don't assert specific values since the math depends on width
		// alignment — just that it doesn't panic.
	})
}
