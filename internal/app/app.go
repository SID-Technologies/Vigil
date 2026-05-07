// Package app is the Vigil sidecar entry point.
package app

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/sid-technologies/vigil/db"
	"github.com/sid-technologies/vigil/internal/aggregator"
	"github.com/sid-technologies/vigil/internal/ipc"
	"github.com/sid-technologies/vigil/internal/monitor"
	"github.com/sid-technologies/vigil/internal/outages"
	"github.com/sid-technologies/vigil/internal/retention"
	"github.com/sid-technologies/vigil/internal/storage"
	"github.com/sid-technologies/vigil/pkg/buildinfo"
	"github.com/sid-technologies/vigil/pkg/errors"
	vlog "github.com/sid-technologies/vigil/pkg/log"
)

// Run executes the sidecar and returns a process exit code.
func Run() int {
	dataDir := flag.String("data-dir", "", "Directory for SQLite DB, logs, and cached settings (required)")
	devMode := flag.Bool("dev", false, "Log to stderr instead of <data-dir>/vigil.log (for `go run`)")

	flag.Parse()

	// Three mutually-exclusive logger setups, ordered by specificity.
	switch {
	case *devMode:
		vlog.InitializeLoggerStderr()
	case *dataDir == "":
		// stdout is reserved for IPC, no log file yet — stderr only.
		_, _ = os.Stderr.WriteString("vigil-sidecar: --data-dir is required (or pass --dev)\n")
		return 2
	default:
		_, err := vlog.InitializeLogger(*dataDir)
		if err != nil {
			_, _ = os.Stderr.WriteString("vigil-sidecar: log init failed: " + err.Error() + "\n")
			return 1
		}
	}

	buildinfo.Instrument()
	log.Info().Str("data_dir", *dataDir).Bool("dev", *devMode).Msg("vigil sidecar starting")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-sigCh
		log.Info().Msg("signal received, shutting down")
		cancel()
	}()

	// Tauri parent dies → we reparent to PID 1 → exit ourselves to avoid zombies.
	go watchParent(ctx, cancel)

	client, err := db.Open(ctx, *dataDir)
	if err != nil {
		log.Error().Err(err).Msg("failed to open database")
		return 1
	}

	defer func() { _ = client.Close() }()

	store := storage.NewClient(client)

	err = store.Seed.DefaultTargets(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to seed default targets")
		return 1
	}

	err = store.Seed.AppConfig(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to seed app_config")
		return 1
	}

	cfg, err := store.Config.Get(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to load app_config")
		return 1
	}

	probeList, err := store.Targets.ListEnabledProbes(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to load enabled probes")
		return 1
	}

	mon := monitor.New(client, monitor.Config{
		PingIntervalSec:   cfg.PingIntervalSec,
		FlushIntervalSec:  cfg.FlushIntervalSec,
		PingTimeoutMs:     cfg.PingTimeoutMs,
		WifiSampleEnabled: cfg.WifiSampleEnabled,
	})

	srv := ipc.NewServer(os.Stdin, os.Stdout)
	ipc.RegisterCoreHandlers(srv)
	ipc.RegisterTargetHandlers(srv, store)
	ipc.RegisterSampleHandlers(srv, store)
	ipc.RegisterWifiHandlers(srv, store)
	ipc.RegisterConfigHandlers(srv, store, func(c storage.AppConfig) {
		mon.UpdateConfig(monitor.Config{
			PingIntervalSec:   c.PingIntervalSec,
			FlushIntervalSec:  c.FlushIntervalSec,
			PingTimeoutMs:     c.PingTimeoutMs,
			WifiSampleEnabled: c.WifiSampleEnabled,
		})
		log.Info().
			Float64("ping_interval_sec", c.PingIntervalSec).
			Int("flush_interval_sec", c.FlushIntervalSec).
			Int("ping_timeout_ms", c.PingTimeoutMs).
			Bool("wifi_sample_enabled", c.WifiSampleEnabled).
			Msg("monitor config hot-reloaded")
	})
	ipc.RegisterOutageHandlers(srv, store)
	ipc.RegisterReportHandlers(srv, store)

	// Tauri 2 event-name validator rejects `.` — only [-/:_] allowed.
	detector := outages.New(client, func(name string, data any) {
		srv.Emit(name, data)
	})
	probeList = mon.AddDynamicGatewayProbe(probeList)
	mon.SetProbes(probeList)

	// Wire after detector + srv exist; detector mutates DB, srv relays to UI.
	mon.SetOnCycle(func(ev monitor.CycleEvent) {
		srv.Emit("probe:cycle", ev)
		detector.OnCycle(ctx, ev)
	})

	agg := aggregator.New(client)
	pruner := retention.New(client, store)

	// wg.Wait blocks the deferred client.Close until every worker has stopped.
	var wg sync.WaitGroup

	wg.Go(func() {
		mon.Run(ctx)
	})

	wg.Go(func() {
		agg.Run(ctx)
	})

	wg.Go(func() {
		pruner.Run(ctx)
	})

	wg.Go(func() {
		err := srv.Run(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			log.Error().Err(err).Msg("ipc server stopped with error")
			cancel()
		}
	})

	wg.Wait()
	log.Info().Msg("vigil sidecar exited cleanly")

	return 0
}

// watchParent cancels ctx when the original parent PID changes (Tauri died and we got reparented).
func watchParent(ctx context.Context, cancel context.CancelFunc) {
	startParent := os.Getppid()
	if startParent <= 1 {
		return
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if os.Getppid() != startParent {
				log.Warn().Int("original_ppid", startParent).Int("current_ppid", os.Getppid()).Msg("parent died, sidecar exiting")
				cancel()

				return
			}
		}
	}
}
