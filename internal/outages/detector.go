// Package outages detects reachability gaps live as probe results stream in
// and persists them as Outage rows.
//
// State machine, per scope:
//
//	idle (consecutive==0, no outageID)
//	  └─ failure → counting (consecutive==1..2)
//	counting
//	  ├─ success → idle
//	  └─ failure ≥3 → open (writes Outage row OR coalesces with recently-closed)
//	open (outageID set, recoveringStartedMs == 0)
//	  ├─ success → recovering (recoveringStartedMs = ts)
//	  └─ failure → updates row (consecutive, errors)
//	recovering (outageID set, recoveringStartedMs > 0)
//	  ├─ success ≥ RecoveryHoldOffMs since recoveringStartedMs → close → idle
//	  ├─ success < RecoveryHoldOffMs → stay recovering, outage row stays open
//	  └─ failure → cancel recovery, back to open, update row
//
// The 3-failure gate keeps single bad probes from generating noise.
// Recovery hold-off prevents one or two lucky probes from declaring "all
// clear" on a flapping connection — the outage row stays open until the
// network has been clean for RecoveryHoldOffMs.
//
// Coalescing: when a new failure run hits the threshold within
// CoalesceWindowMs of a prior closure, the prior outage row is reopened
// (end_ts cleared, errors merged, consecutive_failures cumulative) instead
// of writing a new row. A flapping incident reads as one continuous outage
// rather than dozens of fragments.
//
// Network scope updates once per cycle from "did any probe succeed?".
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
// becomes an Outage row.
const MinConsecutiveFailures = 3

// RecoveryHoldOffMs is how long a scope must be probe-clean before an open
// outage closes. Prevents a single lucky probe from prematurely declaring
// recovery on a flapping connection.
const RecoveryHoldOffMs int64 = 60_000

// CoalesceWindowMs is the lookback after a closure during which a new
// failure run merges into the prior outage row instead of opening a new
// one. Tunes how aggressively flapping incidents fold into one record.
const CoalesceWindowMs int64 = 120_000

type scopeState struct {
	// Counting toward MinConsecutiveFailures. Reset on success when no
	// outage is open. Carried into the row at open time.
	startTSMs   int64
	consecutive int
	errors      map[string]int

	// Non-empty once an Outage row exists for the current incident.
	outageID string

	// Set when the first success arrives during an open outage. The row
	// closes once tsMs - recoveringStartedMs >= RecoveryHoldOffMs. A
	// failure during this window resets it to 0 and the outage stays open.
	recoveringStartedMs int64

	// Set when an outage closes. Used to coalesce a new failure run that
	// hits MinConsecutiveFailures within CoalesceWindowMs back into the
	// prior row.
	lastClosedID    string
	lastClosedTSMs  int64
}

// Detector consumes monitor.CycleEvent payloads and writes/updates Outage
// rows. OnCycle is safe to call from any goroutine.
type Detector struct {
	client *ent.Client

	mu     sync.Mutex
	scopes map[string]*scopeState

	onEvent func(name string, data any)
}

// New builds a Detector. Pass onEvent=nil to disable event emission.
func New(client *ent.Client, onEvent func(name string, data any)) *Detector {
	return &Detector{
		client:  client,
		scopes:  make(map[string]*scopeState),
		onEvent: onEvent,
	}
}

// OnCycle is the monitor's per-cycle callback.
func (d *Detector) OnCycle(ctx context.Context, ev monitor.CycleEvent) {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, r := range ev.Results {
		d.advance(ctx, "target:"+r.Target.Label, r.TimestampMs, r.Success, r.Error)
	}

	// Network scope: failed iff every probe in the cycle failed.
	networkFailed := ev.Total > 0 && ev.OK == 0
	d.advance(ctx, "network", ev.TSUnixMs, !networkFailed, combinedError(ev.Results))
}

func (d *Detector) advance(ctx context.Context, scope string, tsMs int64, success bool, errPtr *string) {
	state, ok := d.scopes[scope]
	if !ok {
		state = &scopeState{}
		d.scopes[scope] = state
	}

	if success {
		d.handleSuccess(ctx, scope, state, tsMs)
		return
	}

	d.handleFailure(ctx, scope, state, tsMs, errPtr)
}

func (d *Detector) handleSuccess(ctx context.Context, scope string, state *scopeState, tsMs int64) {
	if state.outageID == "" {
		// No active outage. Reset any in-progress fail counting.
		state.startTSMs = 0
		state.consecutive = 0
		state.errors = nil

		return
	}

	// Outage is open. First success kicks off the recovery window;
	// subsequent successes hold until the window elapses.
	if state.recoveringStartedMs == 0 {
		state.recoveringStartedMs = tsMs
		return
	}

	if tsMs-state.recoveringStartedMs < RecoveryHoldOffMs {
		return
	}

	closedID := state.outageID

	d.closeOutage(ctx, scope, state, tsMs)

	state.lastClosedID = closedID
	state.lastClosedTSMs = tsMs
	state.recoveringStartedMs = 0
	state.startTSMs = 0
	state.consecutive = 0
	state.errors = nil
	state.outageID = ""
}

func (d *Detector) handleFailure(ctx context.Context, scope string, state *scopeState, tsMs int64, errPtr *string) {
	errKey := "unknown"
	if errPtr != nil {
		errKey = *errPtr
	}

	if state.outageID != "" {
		// Failure during an open (or recovering) outage. Cancel any
		// in-progress recovery, append to the existing row.
		state.recoveringStartedMs = 0
		state.consecutive++

		if state.errors == nil {
			state.errors = map[string]int{}
		}

		state.errors[errKey]++

		d.updateOpenOutage(ctx, state)

		return
	}

	if state.consecutive == 0 {
		state.startTSMs = tsMs
		state.errors = map[string]int{}
	}

	state.consecutive++
	state.errors[errKey]++

	if state.consecutive < MinConsecutiveFailures {
		return
	}

	// Threshold reached. Coalesce with a recently-closed outage if one
	// exists within the window — otherwise open a fresh row.
	if state.lastClosedID != "" && tsMs-state.lastClosedTSMs <= CoalesceWindowMs {
		d.coalesceOutage(ctx, scope, state)
		return
	}

	d.openOutage(ctx, scope, state)
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

// coalesceOutage reopens a recently-closed outage row instead of writing a
// new one. Clears end_ts, merges the new failure run's errors into the
// existing tally, and accumulates consecutive_failures across the merged
// incident. Falls back to opening a fresh row if the prior row can't be
// fetched or updated.
func (d *Detector) coalesceOutage(ctx context.Context, scope string, state *scopeState) {
	priorID := state.lastClosedID

	prior, err := d.client.Outage.Get(ctx, priorID)
	if err != nil {
		log.Warn().Err(err).Str("id", priorID).Msg("outages: coalesce fetch failed; opening new row")
		d.openOutage(ctx, scope, state)

		return
	}

	merged := mergeErrorMaps(prior.Errors, state.errors)
	cumulative := prior.ConsecutiveFailures + state.consecutive

	row, err := d.client.Outage.UpdateOneID(priorID).
		ClearEndTsUnixMs().
		SetConsecutiveFailures(cumulative).
		SetErrors(merged).
		Save(ctx)
	if err != nil {
		log.Error().Err(err).Str("id", priorID).Msg("outages: coalesce update failed")
		return
	}

	state.outageID = priorID
	state.consecutive = cumulative
	state.errors = copyMap(merged)
	state.lastClosedID = ""
	state.lastClosedTSMs = 0

	log.Warn().Str("scope", scope).Str("id", priorID).Int("cumulative", cumulative).
		Msg("outage reopened (coalesced)")

	if d.onEvent != nil {
		d.onEvent("outage:start", outageRowPayloadOf(row))
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

	b, err := json.Marshal(errs)
	if err != nil {
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

func mergeErrorMaps(a, b map[string]int) map[string]int {
	out := make(map[string]int, len(a)+len(b))
	maps.Copy(out, a)

	for k, v := range b {
		out[k] += v
	}

	return out
}

// outageRowPayloadOf shapes an Outage row for outage:start / outage:end events.
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
