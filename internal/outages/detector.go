// Package outages detects reachability gaps live as probe results stream in
// and persists them as Outage rows.
//
// State machine, per scope:
//
//	idle (consecutive==0)
//	  └─ failure → counting (consecutive==1..2)
//	counting
//	  ├─ success → idle
//	  └─ failure ≥3 → open (writes Outage row, sets outageID)
//	open
//	  ├─ success → close (sets end_ts) → idle
//	  └─ failure → updates row (consecutive, errors)
//
// The 3-failure gate keeps single bad probes from generating noise.
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

type scopeState struct {
	startTSMs   int64
	consecutive int
	errors      map[string]int
	outageID    string // non-empty once an Outage row exists
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
		if state.consecutive >= MinConsecutiveFailures && state.outageID != "" {
			d.closeOutage(ctx, scope, state, tsMs)
		}

		state.startTSMs = 0
		state.consecutive = 0
		state.errors = nil
		state.outageID = ""

		return
	}

	if state.consecutive == 0 {
		state.startTSMs = tsMs
		state.errors = map[string]int{}
	}

	state.consecutive++

	errKey := "unknown"
	if errPtr != nil {
		errKey = *errPtr
	}

	state.errors[errKey]++

	if state.consecutive == MinConsecutiveFailures && state.outageID == "" {
		d.openOutage(ctx, scope, state)
	} else if state.consecutive > MinConsecutiveFailures && state.outageID != "" {
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
