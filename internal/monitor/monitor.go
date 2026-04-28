// Package monitor owns the probe loop and flusher.
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

// Config bundles the runtime knobs read from app_config.
type Config struct {
	PingIntervalSec   float64
	FlushIntervalSec  int
	PingTimeoutMs     int
	WifiSampleEnabled bool
}

// CycleEvent is emitted after every probe cycle.
type CycleEvent struct {
	TSUnixMs int64           `json:"ts_unix_ms"`
	Total    int             `json:"total"`
	OK       int             `json:"ok"`
	Fail     int             `json:"fail"`
	Results  []probes.Result `json:"results"`
}

// Monitor runs the probe loop and flusher. Config is hot-reloadable via
// UpdateConfig; subscribers wake on change and re-read at the top of each
// iteration.
type Monitor struct {
	client *ent.Client

	cfgMu sync.RWMutex
	cfg   Config

	// One wake channel per subscribed goroutine. UpdateConfig fans out a
	// non-blocking send to each.
	wakeMu sync.Mutex
	wakers []chan struct{}

	buf     *buffer
	flusher *flusher

	probesMu sync.RWMutex
	probes   []probes.Probe

	onCycle func(CycleEvent)
}

// New constructs a Monitor.
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

// Config returns a copy of the current config.
func (m *Monitor) Config() Config {
	m.cfgMu.RLock()
	defer m.cfgMu.RUnlock()

	return m.cfg
}

// UpdateConfig replaces the running config and wakes all subscribers.
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
			// Pending wake already buffered — coalesce.
		}
	}
}

// SetOnCycle registers a per-cycle callback. Not thread-safe; call once
// before Run.
func (m *Monitor) SetOnCycle(fn func(CycleEvent)) {
	m.onCycle = fn
}

// SetProbes replaces the active probe list.
func (m *Monitor) SetProbes(probeList []probes.Probe) {
	m.probesMu.Lock()
	m.probes = probeList
	m.probesMu.Unlock()
}

// Run blocks until ctx is canceled.
func (m *Monitor) Run(ctx context.Context) {
	var wg sync.WaitGroup

	wg.Go(func() {
		m.flusher.run(ctx)
	})

	wg.Go(func() {
		m.probeLoop(ctx)
	})

	wg.Wait()
}

// AddDynamicGatewayProbe prepends a `router_icmp` probe pointed at the
// auto-detected default gateway. The gateway is resolved fresh on each
// startup — never persisted, since it changes with networks / DHCP.
func (*Monitor) AddDynamicGatewayProbe(existing []probes.Probe) []probes.Probe {
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
		return existing
	}

	return append([]probes.Probe{probe}, existing...)
}

// subscribe returns a capacity-1 wake channel. Cap of 1 holds a wake fired
// between iterations until consumed, so reloads are never missed.
func (m *Monitor) subscribe() <-chan struct{} {
	ch := make(chan struct{}, 1)

	m.wakeMu.Lock()
	m.wakers = append(m.wakers, ch)
	m.wakeMu.Unlock()

	return ch
}

func (m *Monitor) probeLoop(ctx context.Context) {
	wake := m.subscribe()
	cycleStart := time.Now()

	for {
		err := ctx.Err()
		if err != nil {
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
			// Reset cycleStart so the new interval runs from "now," not from
			// the stale schedule.
			cycleStart = time.Now()
			continue
		}

		cycleStart = next
	}
}

func (m *Monitor) runCycle(parent context.Context) {
	m.probesMu.RLock()
	probeList := m.probes
	m.probesMu.RUnlock()

	if len(probeList) == 0 {
		return
	}

	cfg := m.Config()
	cycleTimeout := computeCycleTimeout(cfg.PingTimeoutMs)

	ctx, cancel := context.WithTimeout(parent, cycleTimeout)
	defer cancel()

	results := make([]probes.Result, len(probeList))
	g, gctx := errgroup.WithContext(ctx)
	// Bound concurrency: cap socket count if a user configures thousands of targets.
	g.SetLimit(maxCycleConcurrency)

	for i, p := range probeList {
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
			TSUnixMs: time.Now().UnixMilli(),
			Total:    len(results),
			OK:       ok,
			Fail:     len(results) - ok,
			Results:  results,
		})
	}
}

const maxCycleConcurrency = 64

// computeCycleTimeout derives the per-cycle deadline as 1.5× probeTimeoutMs
// + 500ms (one-retry slack, plus DNS/TLS/scheduler margin).
func computeCycleTimeout(probeTimeoutMs int) time.Duration {
	const (
		retrySlackMultiplier    = 3
		retrySlackDivisor       = 2
		schedulerSafetyMarginMs = 500
	)

	base := time.Duration(probeTimeoutMs) * time.Millisecond

	return base*retrySlackMultiplier/retrySlackDivisor +
		schedulerSafetyMarginMs*time.Millisecond
}
