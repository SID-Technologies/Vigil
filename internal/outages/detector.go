// Package outages detects reachability gaps live as probe results stream in,
// persisting them as Outage rows so the UI can show "outage in progress"
// badges and historical event lists.
//
// Algorithm — direct port of pingscraper.stats.find_target_outages /
// find_full_network_outages, but live (no batch scan):
//
//   - Maintain in-memory state per scope: { startTs, consecutiveFailures, errors }.
//   - On each probe result for a scope:
//     success → if state has 3+ consecutive failures, close the open
//     Outage row by setting end_ts. Reset state.
//     failure → increment counter, capture error, capture startTs if first.
//     When counter hits 3, write a new Outage row with end_ts=null.
//   - Network scope is special: one update per cycle, computed from "did
//     ANY probe succeed in this cycle?".
//
// Why live and not periodic batch:
//   - Frontend gets a real-time `outage:start` event the moment we cross
//     the 3-failure threshold — useful for tray icon color changes.
//   - The 3-consecutive-failures gate means the detector is conservative
//     by construction; one bad probe doesn't generate noise.
package outages

import (
	"context"
	"encoding/json"
	"maps"
	"sync"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/sid-technologies/vigil/db/ent"
	"github.com/sid-technologies/vigil/internal/monitor"
	"github.com/sid-technologies/vigil/internal/probes"
)

// MinConsecutiveFailures is the threshold at which a scope's failure run
// is promoted to an Outage row. Matches the legacy Python value.
const MinConsecutiveFailures = 3

// scopeState tracks an in-flight failure run for a single scope.
// startTSMs records when the run started so we can write it as the Outage's
// start_ts on threshold crossing.
//
// outageID is set once we cross the threshold and write the row — used to
// update end_ts on the closing success.
type scopeState struct {
	startTSMs   int64
	consecutive int
	errors      map[string]int
	outageID    string // non-empty once an Outage row exists
}

// Detector consumes monitor.CycleEvent payloads and writes/updates Outage
// rows. Safe to call OnCycle from any goroutine — internal state is mutex-
// protected.
//
// Event emission: when an outage row is opened, we call onEvent("outage:start").
// When closed, onEvent("outage:end"). Both with the persisted Outage row
// payload so the frontend can update its outage list incrementally.
type Detector struct {
	client *ent.Client

	mu     sync.Mutex
	scopes map[string]*scopeState

	onEvent func(name string, data any)
}

// New builds a Detector that writes to the given Ent client. onEvent is
// optional — pass nil to disable event emission.
func New(client *ent.Client, onEvent func(name string, data any)) *Detector {
	return &Detector{
		client:  client,
		scopes:  make(map[string]*scopeState),
		onEvent: onEvent,
	}
}

// OnCycle is the monitor's per-cycle callback. Wire this into Monitor via
// SetOnCycle alongside the IPC emit.
func (d *Detector) OnCycle(ctx context.Context, ev monitor.CycleEvent) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Per-target scopes.
	for _, r := range ev.Results {
		scope := "target:" + r.Target.Label
		d.advance(ctx, scope, r.TimestampMs, r.Success, r.Error)
	}

	// Network-wide scope: failed iff every probe in the cycle failed.
	networkFailed := ev.Total > 0 && ev.OK == 0
	combinedErr := combinedError(ev.Results)
	d.advance(ctx, "network", ev.TSUnixMs, !networkFailed, combinedErr)
}

// advance is the per-(scope, sample) state machine. Called for every probe
// result and once per cycle for network scope.
//
//nolint:revive // success is the per-sample outcome, not a config flag — splitting would fragment the state machine
func (d *Detector) advance(ctx context.Context, scope string, tsMs int64, success bool, errPtr *string) {
	state, ok := d.scopes[scope]
	if !ok {
		state = &scopeState{}
		d.scopes[scope] = state
	}

	if success {
		// Close any open outage row first, before resetting state.
		if state.consecutive >= MinConsecutiveFailures && state.outageID != "" {
			d.closeOutage(ctx, scope, state, tsMs)
		}

		state.startTSMs = 0
		state.consecutive = 0
		state.errors = nil
		state.outageID = ""

		return
	}

	// Failure path.
	if state.consecutive == 0 {
		state.startTSMs = tsMs
		state.errors = map[string]int{}
	}

	state.consecutive++
	if errPtr != nil {
		state.errors[*errPtr]++
	} else {
		state.errors["unknown"]++
	}

	// Threshold crossing: open a new outage row exactly once.
	if state.consecutive == MinConsecutiveFailures && state.outageID == "" {
		d.openOutage(ctx, scope, state)
	} else if state.consecutive > MinConsecutiveFailures && state.outageID != "" {
		// Update consecutive_failures + errors on the existing row so the
		// row stays current while ongoing.
		d.updateOpenOutage(ctx, state)
	}
}

func (d *Detector) openOutage(ctx context.Context, scope string, state *scopeState) {
	id := uuid.NewString()

	row, err := d.client.Outage.Create().
		SetID(id).
		SetScope(scope).
		SetStartTsUnixMs(state.startTSMs).
		SetConsecutiveFailures(state.consecutive).
		SetErrors(copyMap(state.errors)).
		Save(ctx)
	if err != nil {
		log.Error().Err(err).Str("scope", scope).Msg("outages: open failed")
		return
	}

	state.outageID = id

	log.Warn().Str("scope", scope).Int("consecutive", state.consecutive).Msg("outage detected")

	if d.onEvent != nil {
		d.onEvent("outage:start", outageRowPayloadOf(row))
	}
}

func (d *Detector) updateOpenOutage(ctx context.Context, state *scopeState) {
	_, err := d.client.Outage.UpdateOneID(state.outageID).
		SetConsecutiveFailures(state.consecutive).
		SetErrors(copyMap(state.errors)).
		Save(ctx)
	if err != nil {
		log.Error().Err(err).Str("id", state.outageID).Msg("outages: update failed")
	}
}

func (d *Detector) closeOutage(ctx context.Context, scope string, state *scopeState, endTSMs int64) {
	row, err := d.client.Outage.UpdateOneID(state.outageID).
		SetEndTsUnixMs(endTSMs).
		Save(ctx)
	if err != nil {
		log.Error().Err(err).Str("id", state.outageID).Msg("outages: close failed")
		return
	}

	log.Info().Str("scope", scope).Int("consecutive", state.consecutive).Msg("outage cleared")

	if d.onEvent != nil {
		d.onEvent("outage:end", outageRowPayloadOf(row))
	}
}

func combinedError(results []probes.Result) *string {
	errs := map[string]bool{}

	for _, r := range results {
		if r.Success {
			continue
		}

		if r.Error != nil {
			errs[*r.Error] = true
		}
	}

	if len(errs) == 0 {
		return nil
	}
	// Stable JSON-marshalable summary.
	b, err := json.Marshal(errs)
	if err != nil {
		// map[string]bool can't realistically fail to marshal; log defensively.
		log.Warn().Err(err).Msg("outages: combined-error marshal failed")
		return nil
	}

	s := string(b)

	return &s
}

func copyMap(m map[string]int) map[string]int {
	out := make(map[string]int, len(m))
	maps.Copy(out, m)

	return out
}

// outageRowPayloadOf is the JSON shape emitted to the frontend on
// outage:start / outage:end events. Mirrors storage.Outage exactly so the
// frontend has one type definition.
func outageRowPayloadOf(r *ent.Outage) map[string]any {
	return map[string]any{
		"id":                   r.ID,
		"scope":                r.Scope,
		"start_ts_unix_ms":     r.StartTsUnixMs,
		"end_ts_unix_ms":       r.EndTsUnixMs,
		"consecutive_failures": r.ConsecutiveFailures,
		"errors":               r.Errors,
	}
}
