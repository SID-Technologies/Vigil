// Package app orchestrates the Vigil sidecar's lifetime. The sidecar is a
// single long-running process spawned by the Tauri shell. It:
//
//  1. Initializes file-based logging under --data-dir
//  2. Watches the parent PID and exits if Tauri dies (orphan protection)
//  3. Opens the SQLite DB via Ent, runs schema migration, seeds defaults
//  4. Loads app_config and starts the monitor goroutines
//  5. Wires the stdio JSON-RPC IPC server on stdin/stdout
//  6. Blocks until the parent closes stdin (clean shutdown) or SIGTERM/SIGINT
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

// Run is the sidecar entry point. Returns a process exit code.
func Run() int {
	dataDir := flag.String("data-dir", "", "Directory for SQLite DB, logs, and cached settings (required)")
	devMode := flag.Bool("dev", false, "Log to stderr instead of <data-dir>/vigil.log (for `go run`)")

	flag.Parse() //nolint:revive // app.Run is the sidecar's effective main; cmd/vigil-sidecar just delegates here

	if *devMode {
		vlog.InitializeLoggerStderr()
	} else {
		if *dataDir == "" {
			// Can't log to file yet, can't log to stdout (IPC). Stderr is the
			// only safe channel.
			_, _ = os.Stderr.WriteString("vigil-sidecar: --data-dir is required (or pass --dev)\n")
			return 2
		}

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

	// SIGTERM/SIGINT — graceful shutdown.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-sigCh
		log.Info().Msg("signal received, shutting down")
		cancel()
	}()

	// Parent-PID watcher — if Tauri dies without cleaning up, exit ourselves.
	go watchParent(ctx, cancel)

	// Storage layer.
	client, err := db.Open(ctx, *dataDir)
	if err != nil {
		log.Error().Err(err).Msg("failed to open database")
		return 1
	}

	defer func() { _ = client.Close() }()

	err = storage.SeedDefaultTargets(ctx, client)
	if err != nil {
		log.Error().Err(err).Msg("failed to seed default targets")
		return 1
	}

	err = storage.SeedAppConfig(ctx, client)
	if err != nil {
		log.Error().Err(err).Msg("failed to seed app_config")
		return 1
	}

	cfg, err := storage.GetAppConfig(ctx, client)
	if err != nil {
		log.Error().Err(err).Msg("failed to load app_config")
		return 1
	}

	// Monitor — built early so the IPC config handler can hot-reload it.
	probeList, err := storage.ListEnabledProbes(ctx, client)
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

	// IPC server.
	srv := ipc.NewServer(os.Stdin, os.Stdout)
	ipc.RegisterCoreHandlers(srv)
	ipc.RegisterTargetHandlers(srv, client)
	ipc.RegisterSampleHandlers(srv, client)
	ipc.RegisterWifiHandlers(srv, client)
	ipc.RegisterConfigHandlers(srv, client, func(c storage.AppConfig) {
		// Hot-reload — the monitor's running probe loop and flusher pick up
		// the new values on their next iteration. No sidecar restart needed.
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
	ipc.RegisterOutageHandlers(srv, client)
	ipc.RegisterReportHandlers(srv, client)

	// Outage detector — fed by the monitor's per-cycle callback.
	detector := outages.New(client, func(name string, data any) {
		// Note: Tauri 2 event-name validator rejects `.` — only [-/:_] allowed.
		srv.Emit(name, data)
	})
	probeList = mon.AddDynamicGatewayProbe(probeList)
	mon.SetProbes(probeList)

	// Wire the cycle event AFTER detector + srv exist. Both consume the same
	// per-cycle event; the detector mutates DB state, the IPC emit relays
	// to the live UI.
	mon.SetOnCycle(func(ev monitor.CycleEvent) {
		// Forward to the frontend AND feed the outage detector. Both consume
		// the same per-cycle event; the detector mutates DB state, the IPC
		// emit just relays for the live UI.
		srv.Emit("probe:cycle", ev)
		detector.OnCycle(ctx, ev)
	})

	// Aggregator + retention pruner — both run independently of monitor cadence.
	agg := aggregator.New(client)
	pruner := retention.New(client)

	// Run all four background workers + IPC server. wg ensures we don't return
	// (and trigger client.Close via defer) until everyone has stopped.
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

// watchParent polls the parent PID. When the parent (Tauri shell) dies, this
// process is reparented to PID 1 (or the equivalent on Windows). Detecting
// that and canceling ctx ensures the sidecar doesn't outlive its host as a
// zombie consuming CPU.
func watchParent(ctx context.Context, cancel context.CancelFunc) {
	startParent := os.Getppid()
	if startParent <= 1 {
		// We were already orphaned at startup, or started directly from a
		// shell. Skip the watcher.
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
