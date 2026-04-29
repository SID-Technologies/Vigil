package storage_test

import (
	"context"
	"testing"

	"github.com/sid-technologies/vigil/db"
	"github.com/sid-technologies/vigil/internal/probes"
	"github.com/sid-technologies/vigil/internal/storage"
)

func setup(t *testing.T) (context.Context, *storage.Client) {
	t.Helper()

	client := db.SetupTestEntClientDB(t)

	return context.Background(), storage.NewClient(client)
}

// ----------------------------------------------------------------------------
// Targets
// ----------------------------------------------------------------------------

func TestTargets_CreateListGetUpdateDelete(t *testing.T) {
	t.Parallel()

	ctx, store := setup(t)
	port := 53

	created, err := store.Targets.Create(ctx, storage.TargetRequest{
		Label: "1.1.1.1_dns",
		Kind:  probes.KindUDPDNS,
		Host:  "1.1.1.1",
		Port:  &port,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if created.IsBuiltin {
		t.Error("user-created target should be is_builtin=false")
	}

	if !created.Enabled {
		t.Error("user-created target should default to enabled")
	}

	list, err := store.Targets.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	if len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("list returned %+v", list)
	}

	got, err := store.Targets.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if got.ID != created.ID {
		t.Fatal("get returned wrong row")
	}

	disabled := false

	updated, err := store.Targets.Update(ctx, created.ID, storage.TargetUpdateRequest{
		Enabled: &disabled,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	if updated.Enabled {
		t.Error("update did not persist Enabled=false")
	}

	err = store.Targets.Delete(ctx, created.ID)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}

	list, err = store.Targets.List(ctx)
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}

	if len(list) != 0 {
		t.Fatalf("delete left %d rows behind", len(list))
	}
}

// ListEnabledProbes must skip disabled rows and convert the rest into runnable probes.
func TestTargets_ListEnabledProbesSkipsDisabled(t *testing.T) {
	t.Parallel()

	ctx, store := setup(t)
	port := 53

	enabled, err := store.Targets.Create(ctx, storage.TargetRequest{
		Label: "on", Kind: probes.KindUDPDNS, Host: "1.1.1.1", Port: &port,
	})
	if err != nil {
		t.Fatal(err)
	}

	disabledTarget, err := store.Targets.Create(ctx, storage.TargetRequest{
		Label: "off", Kind: probes.KindICMP, Host: "8.8.8.8",
	})
	if err != nil {
		t.Fatal(err)
	}

	disabled := false

	_, err = store.Targets.Update(ctx, disabledTarget.ID, storage.TargetUpdateRequest{
		Enabled: &disabled,
	})
	if err != nil {
		t.Fatal(err)
	}

	probesList, err := store.Targets.ListEnabledProbes(ctx)
	if err != nil {
		t.Fatalf("list enabled probes: %v", err)
	}

	if len(probesList) != 1 {
		t.Fatalf("got %d probes, want 1", len(probesList))
	}

	if probesList[0].Target().Label != enabled.Label {
		t.Fatalf("wrong probe surfaced: %+v", probesList[0].Target())
	}
}

// ----------------------------------------------------------------------------
// Seed
// ----------------------------------------------------------------------------

// DefaultTargets seeds builtin probe targets only when the table is empty.
// A user who deletes every default should not see them resurrected on next boot.
func TestSeed_DefaultTargetsSkipsWhenAnyExists(t *testing.T) {
	t.Parallel()

	ctx, store := setup(t)

	err := store.Seed.DefaultTargets(ctx)
	if err != nil {
		t.Fatalf("first seed: %v", err)
	}

	first, err := store.Targets.List(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(first) == 0 {
		t.Fatal("first seed produced no rows")
	}

	for _, target := range first {
		if !target.IsBuiltin {
			t.Errorf("seeded target %s should be builtin", target.Label)
		}
	}

	// Re-seed should be a no-op.
	err = store.Seed.DefaultTargets(ctx)
	if err != nil {
		t.Fatalf("re-seed: %v", err)
	}

	second, err := store.Targets.List(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(second) != len(first) {
		t.Fatalf("re-seed mutated row count: %d → %d", len(first), len(second))
	}
}

// AppConfig seeds the singleton config row exactly once.
func TestSeed_AppConfigIdempotent(t *testing.T) {
	t.Parallel()

	ctx, store := setup(t)

	err := store.Seed.AppConfig(ctx)
	if err != nil {
		t.Fatalf("first seed: %v", err)
	}

	cfg, err := store.Config.Get(ctx)
	if err != nil {
		t.Fatalf("config get: %v", err)
	}

	if cfg.PingIntervalSec <= 0 {
		t.Fatalf("seeded config has invalid PingIntervalSec=%v", cfg.PingIntervalSec)
	}

	err = store.Seed.AppConfig(ctx)
	if err != nil {
		t.Fatalf("re-seed: %v", err)
	}

	cfg2, err := store.Config.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if cfg != cfg2 {
		t.Fatalf("re-seed mutated config: %+v → %+v", cfg, cfg2)
	}
}

// ----------------------------------------------------------------------------
// Config
// ----------------------------------------------------------------------------

func TestConfig_PartialPatchLeavesOtherFields(t *testing.T) {
	t.Parallel()

	ctx, store := setup(t)

	err := store.Seed.AppConfig(ctx)
	if err != nil {
		t.Fatal(err)
	}

	before, err := store.Config.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}

	newInterval := 5.0
	patch := storage.AppConfigPatch{PingIntervalSec: &newInterval}

	got, err := store.Config.Update(ctx, patch)
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	if got.PingIntervalSec != newInterval {
		t.Errorf("PingIntervalSec = %v, want %v", got.PingIntervalSec, newInterval)
	}

	// Every other field must equal its pre-patch value.
	if got.FlushIntervalSec != before.FlushIntervalSec ||
		got.PingTimeoutMs != before.PingTimeoutMs ||
		got.RetentionRawDays != before.RetentionRawDays ||
		got.Retention1minDays != before.Retention1minDays ||
		got.Retention5minDays != before.Retention5minDays ||
		got.WifiSampleEnabled != before.WifiSampleEnabled {
		t.Errorf("partial patch leaked into other fields: before=%+v after=%+v", before, got)
	}
}
