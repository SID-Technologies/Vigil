package aggregator_test

import (
	"context"
	"testing"
	"time"

	"github.com/sid-technologies/vigil/db"
	"github.com/sid-technologies/vigil/db/ent"
	"github.com/sid-technologies/vigil/db/ent/sample1min"
	"github.com/sid-technologies/vigil/internal/aggregator"
	"github.com/sid-technologies/vigil/internal/storage"
)

// TestAggregator_OneMinTier_SixHourWindow seeds 6h of raw probes at the
// real cadence, runs the aggregator, and asserts the 1-min tier ends up
// with one bucket per target per minute — the shape History expects when
// it switches to the 1-min granularity for ranges between 30min and 6h.
func TestAggregator_OneMinTier_SixHourWindow(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := db.SetupTestEntClientDB(t)

	// Anchor "now" past the safety margin so all six hours of buckets are
	// considered closed. Floor to a minute so timestamps line up cleanly
	// for assertions.
	const targetLabel = "router_icmp"

	const probeIntervalMs = int64(2_500)

	const sixHoursMs = int64(6 * 60 * 60 * 1000)

	nowMs := (time.Now().UnixMilli() / aggregator.OneMinMs) * aggregator.OneMinMs

	// Generate 6h of raw probes at 2.5s cadence — one target, all successful.
	startMs := nowMs - sixHoursMs - 2*aggregator.SafetyMarginMs
	endMs := nowMs - 2*aggregator.SafetyMarginMs

	// SQLite caps bound vars per INSERT — chunk the bulk create.
	const batchSize = 500

	batch := make([]*ent.SampleCreate, 0, batchSize)
	flush := func() {
		if len(batch) == 0 {
			return
		}

		_, err := client.Sample.CreateBulk(batch...).Save(ctx)
		if err != nil {
			t.Fatalf("seed raw samples: %v", err)
		}

		batch = batch[:0]
	}

	for ts := startMs; ts < endMs; ts += probeIntervalMs {
		rtt := 12.5

		batch = append(batch, client.Sample.Create().
			SetTsUnixMs(ts).
			SetTargetLabel(targetLabel).
			SetTargetKind("icmp").
			SetTargetHost("192.168.1.1").
			SetSuccess(true).
			SetRttMs(rtt))

		if len(batch) >= batchSize {
			flush()
		}
	}

	flush()

	agg := aggregator.New(client)
	// Bump the lookback so a single run sees the entire 6h window. Default is
	// already 6h but the safety margin shaves the trailing edge off — give it
	// real headroom.
	agg.Lookback1minMs = sixHoursMs + aggregator.OneHourMs

	// Drive a single aggregation pass directly, no goroutine.
	agg.RunOnce(ctx)

	// All bucket starts must be exact 60_000ms multiples.
	rows, err := client.Sample1min.Query().
		Where(sample1min.TargetLabelEQ(targetLabel)).
		Order(ent.Asc(sample1min.FieldBucketStartUnixMs)).
		All(ctx)
	if err != nil {
		t.Fatalf("read 1-min buckets: %v", err)
	}

	if len(rows) == 0 {
		t.Fatal("expected 1-min buckets to be written; got 0")
	}

	for _, r := range rows {
		if r.BucketStartUnixMs%aggregator.OneMinMs != 0 {
			t.Fatalf("bucket_start %d is not a 60_000ms multiple", r.BucketStartUnixMs)
		}
	}

	// Now query through the same path History uses (storage.SampleClient.Query1Min)
	// over a 6h window ending at "now" — should match the rows we just wrote.
	store := storage.NewSampleClient(client)

	queryFromMs := nowMs - sixHoursMs

	queryRows, err := store.Query1Min(ctx, storage.QueryAggregatedParams{
		FromMs:       queryFromMs,
		ToMs:         nowMs,
		TargetLabels: []string{targetLabel},
	})
	if err != nil {
		t.Fatalf("Query1Min: %v", err)
	}

	if len(queryRows) == 0 {
		t.Fatal("Query1Min returned 0 rows for a 6h window")
	}

	// Should land in the legibility band: 1-min buckets across 6h = 360, give
	// or take edge truncation. A drop into the dozens would mean the
	// aggregator stitched buckets at the wrong width.
	const minExpected = 300

	if len(queryRows) < minExpected {
		t.Errorf("Query1Min returned %d rows over 6h; want >= %d", len(queryRows), minExpected)
	}

	// Successful probes only → every bucket has p50 set.
	for _, r := range queryRows {
		if r.RTTP50Ms == nil {
			t.Errorf("bucket %d missing p50", r.BucketStartUnixMs)
		}

		if r.SuccessCount == 0 {
			t.Errorf("bucket %d has zero successes", r.BucketStartUnixMs)
		}
	}
}
