package outages_test

import (
	"context"
	"testing"

	"github.com/sid-technologies/vigil/db"
	"github.com/sid-technologies/vigil/db/ent"
	"github.com/sid-technologies/vigil/db/ent/outage"
	"github.com/sid-technologies/vigil/internal/monitor"
	"github.com/sid-technologies/vigil/internal/outages"
	"github.com/sid-technologies/vigil/internal/probes"
)

// Each test sets up an isolated in-memory SQLite DB via SetupTestEntClientDB
// and walks the detector through a sequence of synthetic probe cycles,
// asserting on the resulting Outage rows after each step.

func TestDetector_OpensAfterThreeFailures(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := db.SetupTestEntClientDB(t)
	d := outages.New(client, nil)

	// Two failures: should NOT open an outage yet.
	d.OnCycle(ctx, makeCycle(t, 1000, []probeOutcome{{label: "router_icmp", success: false, err: "timeout"}}))
	d.OnCycle(ctx, makeCycle(t, 2000, []probeOutcome{{label: "router_icmp", success: false, err: "timeout"}}))
	assertOutageCount(ctx, t, client, "target:router_icmp", 0)

	// Third failure crosses the threshold — outage row written.
	d.OnCycle(ctx, makeCycle(t, 3000, []probeOutcome{{label: "router_icmp", success: false, err: "timeout"}}))
	assertOutageCount(ctx, t, client, "target:router_icmp", 1)

	// Verify the persisted outage shape.
	row := mustGetOnly(ctx, t, client, "target:router_icmp")
	if row.StartTsUnixMs != 1000 {
		t.Errorf("start_ts = %d, want 1000 (first failure)", row.StartTsUnixMs)
	}

	if row.EndTsUnixMs != nil {
		t.Errorf("end_ts should be nil while ongoing, got %v", row.EndTsUnixMs)
	}

	if row.ConsecutiveFailures != 3 {
		t.Errorf("consecutive = %d, want 3", row.ConsecutiveFailures)
	}

	if row.Errors["timeout"] != 3 {
		t.Errorf("errors map = %v, want timeout=3", row.Errors)
	}
}

func TestDetector_UpdatesWhileOngoing(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := db.SetupTestEntClientDB(t)
	d := outages.New(client, nil)

	// Drive 5 consecutive failures with mixed errors.
	for i, e := range []string{"timeout", "timeout", "host_unreachable", "timeout", "host_unreachable"} {
		d.OnCycle(ctx, makeCycle(t, int64(i+1)*1000, []probeOutcome{{label: "google_dns_icmp", success: false, err: e}}))
	}

	row := mustGetOnly(ctx, t, client, "target:google_dns_icmp")
	if row.ConsecutiveFailures != 5 {
		t.Errorf("consecutive = %d, want 5", row.ConsecutiveFailures)
	}

	if row.Errors["timeout"] != 3 || row.Errors["host_unreachable"] != 2 {
		t.Errorf("error tally wrong: %v", row.Errors)
	}
}

func TestDetector_ClosesOnSuccess(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := db.SetupTestEntClientDB(t)
	d := outages.New(client, nil)

	// Open the outage.
	for i := 1; i <= 3; i++ {
		d.OnCycle(ctx, makeCycle(t, int64(i)*1000, []probeOutcome{{label: "teams_tcp443", success: false, err: "timeout"}}))
	}

	assertOutageCount(ctx, t, client, "target:teams_tcp443", 1)

	// First success closes it. end_ts is the success cycle's timestamp.
	d.OnCycle(ctx, makeCycle(t, 4000, []probeOutcome{{label: "teams_tcp443", success: true, rtt: 25.5}}))

	row := mustGetOnly(ctx, t, client, "target:teams_tcp443")
	if row.EndTsUnixMs == nil {
		t.Fatal("end_ts not set after close")
	}

	if *row.EndTsUnixMs != 4000 {
		t.Errorf("end_ts = %d, want 4000", *row.EndTsUnixMs)
	}
}

func TestDetector_DoesNotReopenAfterClose(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := db.SetupTestEntClientDB(t)
	d := outages.New(client, nil)

	// Open + close.
	for i := 1; i <= 3; i++ {
		d.OnCycle(ctx, makeCycle(t, int64(i)*1000, []probeOutcome{{label: "x", success: false, err: "timeout"}}))
	}

	d.OnCycle(ctx, makeCycle(t, 4000, []probeOutcome{{label: "x", success: true, rtt: 10}}))

	// One isolated failure should not open a new outage.
	d.OnCycle(ctx, makeCycle(t, 5000, []probeOutcome{{label: "x", success: false, err: "timeout"}}))
	assertOutageCount(ctx, t, client, "target:x", 1)

	// But a fresh run of three failures starts a new outage row.
	d.OnCycle(ctx, makeCycle(t, 6000, []probeOutcome{{label: "x", success: false, err: "timeout"}}))
	d.OnCycle(ctx, makeCycle(t, 7000, []probeOutcome{{label: "x", success: false, err: "timeout"}}))
	assertOutageCount(ctx, t, client, "target:x", 2)
}

func TestDetector_NetworkScopeWhenAllFail(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := db.SetupTestEntClientDB(t)
	d := outages.New(client, nil)

	// All 2 probes fail in 3 consecutive cycles → network outage.
	for i := 1; i <= 3; i++ {
		d.OnCycle(ctx, makeCycle(t, int64(i)*1000, []probeOutcome{
			{label: "a", success: false, err: "timeout"},
			{label: "b", success: false, err: "host_unreachable"},
		}))
	}

	assertOutageCount(ctx, t, client, "network", 1)
}

func TestDetector_NetworkScopeNotTriggeredIfOneSucceeds(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := db.SetupTestEntClientDB(t)
	d := outages.New(client, nil)

	// Even with mostly failures, if one probe succeeds the network is "up".
	for i := 1; i <= 5; i++ {
		d.OnCycle(ctx, makeCycle(t, int64(i)*1000, []probeOutcome{
			{label: "a", success: false, err: "timeout"},
			{label: "b", success: true, rtt: 30},
		}))
	}
	// Per-target outage on `a` should exist, but no network outage.
	assertOutageCount(ctx, t, client, "target:a", 1)
	assertOutageCount(ctx, t, client, "network", 0)
}

// ============================================================================
// Helpers
// ============================================================================

type probeOutcome struct {
	label   string
	success bool
	rtt     float64
	err     string
}

func makeCycle(t *testing.T, tsMs int64, outcomes []probeOutcome) monitor.CycleEvent {
	t.Helper()

	results := make([]probes.Result, len(outcomes))
	ok := 0

	for i, o := range outcomes {
		r := probes.Result{
			TimestampMs: tsMs,
			Target: probes.Target{
				Label: o.label,
				Kind:  probes.KindICMP,
				Host:  "1.1.1.1",
			},
			Success: o.success,
		}
		if o.success {
			rtt := o.rtt
			r.RTTMs = &rtt
			ok++
		} else {
			err := o.err
			r.Error = &err
		}

		results[i] = r
	}

	return monitor.CycleEvent{
		TSUnixMs: tsMs,
		Total:    len(outcomes),
		OK:       ok,
		Fail:     len(outcomes) - ok,
		Results:  results,
	}
}

func assertOutageCount(ctx context.Context, t *testing.T, client *ent.Client, scope string, want int) {
	t.Helper()

	got, err := client.Outage.Query().Where(outage.ScopeEQ(scope)).Count(ctx)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if got != want {
		t.Errorf("outage count for scope=%s = %d, want %d", scope, got, want)
	}
}

func mustGetOnly(ctx context.Context, t *testing.T, client *ent.Client, scope string) *ent.Outage {
	t.Helper()

	rows, err := client.Outage.Query().Where(outage.ScopeEQ(scope)).All(ctx)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if len(rows) != 1 {
		t.Fatalf("expected exactly 1 outage for scope=%s, got %d", scope, len(rows))
	}

	return rows[0]
}
