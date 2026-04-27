// Package monitor owns the probe loop. It:
//
//  1. Loads enabled targets from the DB at startup (and at config-change time
//     in later phases).
//  2. Optionally adds a synthetic `router_icmp` probe pointed at the
//     auto-detected default gateway.
//  3. Fires all probes in parallel each cycle, with a per-probe timeout.
//  4. Pushes results into an in-memory buffer.
//  5. Runs a flusher goroutine that drains the buffer to SQLite every
//     `flush_interval_sec`, capturing a Wi-Fi sample at the same cadence.
//  6. Emits a `probe.cycle` event after each cycle so the frontend can
//     update the live dashboard without polling.
//
// Direct port of pingscraper.monitor.Monitor — same behavior, different
// concurrency primitives (goroutines + channels instead of threads + locks).
package monitor

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"

	"github.com/sid-technologies/vigil/db/ent"
	"github.com/sid-technologies/vigil/internal/netinfo"
	"github.com/sid-technologies/vigil/internal/probes"
)

// Config bundles the runtime knobs read from app_config (or overridden in
// tests). Held by value — the monitor doesn't watch the DB for changes
// within a session; restart applies new values.
type Config struct {
	PingIntervalSec   float64
	FlushIntervalSec  int
	PingTimeoutMs     int
	WifiSampleEnabled bool
}

// CycleEvent is emitted after every probe cycle. The frontend uses this to
// render the live dashboard without polling.
type CycleEvent struct {
	TsUnixMs int64           `json:"ts_unix_ms"`
	Total    int             `json:"total"`
	OK       int             `json:"ok"`
	Fail     int             `json:"fail"`
	Results  []probes.Result `json:"results"`
}

// Monitor runs the probe loop and the flusher. Single Start() call,
// Stop()-via-context-cancel.
//
// Config is hot-reloadable: callers invoke UpdateConfig with a new value
// and the running goroutines pick up the change without a restart. The
// probe loop and flusher each subscribe to a wake channel so their sleeps
// are interrupted on config change — perceived latency for an interval
// shrink is ~0, for a grow it's "wait out the current interval" which is
// the conservative choice anyway.
type Monitor struct {
	client *ent.Client

	cfgMu sync.RWMutex
	cfg   Config

	// Wake channels — one per goroutine that subscribes via subscribe().
	// UpdateConfig sends a wake to every channel (non-blocking).
	wakeMu sync.Mutex
	wakers []chan struct{}

	buf     *buffer
	flusher *flusher

	probesMu sync.RWMutex
	probes   []probes.Probe

	// onCycle is called after each probe cycle completes (any goroutine).
	onCycle func(CycleEvent)
}

// New constructs a Monitor. The probe list is loaded by Start() so the
// monitor can be wired up before storage is fully ready in tests.
func New(client *ent.Client, cfg Config) *Monitor {
	buf := newBuffer()
	m := &Monitor{
		client: client,
		cfg:    cfg,
		buf:    buf,
	}
	m.flusher = newFlusher(client, buf, m)
	return m
}

// Config returns a copy of the current config. Safe to call from any goroutine.
func (m *Monitor) Config() Config {
	m.cfgMu.RLock()
	defer m.cfgMu.RUnlock()
	return m.cfg
}

// UpdateConfig replaces the running config and wakes all subscribers. Called
// by the IPC layer after `config.update` lands a successful DB write.
//
// This is the entry point for hot-reload. The probe loop and flusher each
// re-read config at the top of their loops, so they pick up the change
// within one iteration.
func (m *Monitor) UpdateConfig(cfg Config) {
	m.cfgMu.Lock()
	m.cfg = cfg
	m.cfgMu.Unlock()

	m.wakeMu.Lock()
	defer m.wakeMu.Unlock()
	for _, w := range m.wakers {
		select {
		case w <- struct{}{}:
		default:
			// Channel already has a pending wake — coalescing is fine, the
			// receiver re-reads config either way.
		}
	}
}

// subscribe registers a buffered (capacity 1) wake channel. Goroutines
// select on it alongside their own ticker / timeout to be woken when
// config changes. Capacity 1 means a wake fired between two iterations
// is held until consumed — no missed reloads.
func (m *Monitor) subscribe() <-chan struct{} {
	ch := make(chan struct{}, 1)
	m.wakeMu.Lock()
	m.wakers = append(m.wakers, ch)
	m.wakeMu.Unlock()
	return ch
}

// SetOnCycle registers a callback invoked after each probe cycle. Used by
// the IPC layer to forward `probe.cycle` events. Not thread-safe — call
// once before Start.
func (m *Monitor) SetOnCycle(fn func(CycleEvent)) {
	m.onCycle = fn
}

// SetProbes replaces the active probe list. Called once at startup with
// SeedAndLoadProbes(); will be wired to config changes in later phases.
func (m *Monitor) SetProbes(probeList []probes.Probe) {
	m.probesMu.Lock()
	m.probes = probeList
	m.probesMu.Unlock()
}

// Run blocks until ctx is cancelled. Spawns the probe loop and flusher
// goroutines; ensures both have stopped before returning so the sidecar can
// shut down cleanly.
func (m *Monitor) Run(ctx context.Context) {
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		m.flusher.run(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		m.probeLoop(ctx)
	}()

	wg.Wait()
}

// probeLoop fires all probes in parallel each cycle, sleeping between cycles
// so the start-of-cycle aligns to a fixed cadence (no drift, even if a slow
// cycle pushes us past schedule — we just resync).
//
// Config is re-read every iteration via m.Config() so a change made via
// UpdateConfig takes effect on the very next cycle. The wake channel
// shortens that to "next iteration of this loop" by interrupting sleep.
func (m *Monitor) probeLoop(ctx context.Context) {
	wake := m.subscribe()
	cycleStart := time.Now()
	for {
		if err := ctx.Err(); err != nil {
			log.Info().Msg("monitor: probe loop exiting")
			return
		}
		m.runCycle(ctx)

		interval := time.Duration(m.Config().PingIntervalSec * float64(time.Second))
		next := cycleStart.Add(interval)
		sleep := time.Until(next)
		if sleep <= 0 {
			// Fell behind — resync rather than fire rapidly.
			cycleStart = time.Now()
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(sleep):
		case <-wake:
			// Config changed — restart the loop. cycleStart updates so the
			// next cycle uses the new interval from "now," not from the
			// stale schedule.
			cycleStart = time.Now()
			continue
		}
		cycleStart = next
	}
}

// runCycle fires every probe in parallel with a bounded errgroup. Each probe
// has its own per-call timeout; the cycle as a whole is capped at 1.5× the
// probe timeout so a stalled DNS lookup can't delay the next cycle.
func (m *Monitor) runCycle(parent context.Context) {
	m.probesMu.RLock()
	probeList := m.probes
	m.probesMu.RUnlock()

	if len(probeList) == 0 {
		return
	}

	cfg := m.Config()
	cycleTimeout := time.Duration(cfg.PingTimeoutMs)*time.Millisecond*3/2 + 500*time.Millisecond
	ctx, cancel := context.WithTimeout(parent, cycleTimeout)
	defer cancel()

	results := make([]probes.Result, len(probeList))
	g, gctx := errgroup.WithContext(ctx)
	// Bound concurrency so we don't open a thousand sockets if the user
	// configures a giant target list.
	g.SetLimit(64)

	for i, p := range probeList {
		i, p := i, p
		g.Go(func() error {
			results[i] = p.Run(gctx, cfg.PingTimeoutMs)
			return nil
		})
	}
	_ = g.Wait()

	m.buf.pushMany(results)

	if m.onCycle != nil {
		ok := 0
		for _, r := range results {
			if r.Success {
				ok++
			}
		}
		m.onCycle(CycleEvent{
			TsUnixMs: time.Now().UnixMilli(),
			Total:    len(results),
			OK:       ok,
			Fail:     len(results) - ok,
			Results:  results,
		})
	}
}

// AddDynamicGatewayProbe checks for a default gateway and, if found, prepends
// a synthetic `router_icmp` probe to the active probe list. Called once at
// startup before Run().
//
// The gateway is dynamic (changes with networks / DHCP) so we never persist
// it — it's resolved fresh on each sidecar startup.
func (m *Monitor) AddDynamicGatewayProbe(existing []probes.Probe) []probes.Probe {
	gateway, ok := netinfo.DetectDefaultGateway()
	if !ok {
		log.Warn().Msg("monitor: no default gateway detected — skipping router probe")
		return existing
	}
	log.Info().Str("gateway", gateway).Msg("monitor: detected default gateway")

	router := probes.Target{
		Label: "router_icmp",
		Kind:  probes.KindICMP,
		Host:  gateway,
	}
	probe, err := probes.Build(router)
	if err != nil {
		// Should never happen — KindICMP is exhaustive in factory.go.
		return existing
	}
	return append([]probes.Probe{probe}, existing...)
}
