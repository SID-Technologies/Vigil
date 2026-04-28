import { useMemo, useSyncExternalStore } from 'react';
import {
  getProbeBuffer,
  getProbeCount,
  getProbeStoreVersion,
  subscribeProbeStore,
} from '../lib/probeStore';
import type { ProbeResult } from './useProbeCycle';

/**
 * Aggregated live state for one target — what the tile renders.
 */
export interface LiveTargetState {
  /** All retained probe results in time order, oldest → newest. */
  results: ProbeResult[];
  /** RTTs of successful probes within the window, time order. For sparkline. */
  successfulRTTs: number[];
  /** 0..100, or null if no samples yet. */
  successPct: number | null;
  /** Mean RTT across successful probes in window, ms. null if no successes. */
  avgRTTMs: number | null;
  /** Most recent probe result — used for the tile's instantaneous status. */
  latest: ProbeResult | null;
}

const EMPTY_STATE: LiveTargetState = {
  results: [],
  successfulRTTs: [],
  successPct: null,
  avgRTTMs: null,
  latest: null,
};

/**
 * Per-target rolling-window summaries plus a tick counter for "alive" pulses.
 *
 * The data lives in the singleton probe store (lib/probeStore). This hook is
 * a thin reader: subscribe to the store's version number, then derive
 * summaries from the current buffer. Crucially, the buffer survives route
 * changes — navigating Dashboard → Settings → Dashboard no longer wipes the
 * 5-minute history that powers tile sparklines and success%.
 */
export function useLiveSamples(): {
  states: Map<string, LiveTargetState>;
  /** Increments on each probe:cycle — handy as a render trigger / pulse counter. */
  tick: number;
} {
  const version = useSyncExternalStore(
    subscribeProbeStore,
    getProbeStoreVersion,
    getProbeStoreVersion,
  );

  // Recompute summaries when the store version changes. The buffer reference
  // is stable across calls (the store mutates it in place), so we depend on
  // `version` to drive recomputation.
  const states = useMemo(() => computeStates(getProbeBuffer()), [version]);

  return { states, tick: getProbeCount() };
}

function computeStates(buffer: Map<string, ProbeResult[]>): Map<string, LiveTargetState> {
  const out = new Map<string, LiveTargetState>();
  for (const [label, results] of buffer.entries()) {
    out.set(label, summarize(results));
  }
  return out;
}

export function getLiveState(
  states: Map<string, LiveTargetState>,
  label: string,
): LiveTargetState {
  return states.get(label) ?? EMPTY_STATE;
}

/**
 * Raw access to the probe store's per-target buffer. Used by the
 * dashboard's RTT chart, which needs every probe — not just the
 * tile-summary fields — and wants to update on every cycle without a
 * DB round-trip. Returns the buffer reference (stable; the store mutates
 * in place) plus a version that bumps on each cycle, so callers can
 * `useMemo` against it.
 */
export function useLiveProbes(): {
  buffer: Map<string, ProbeResult[]>;
  version: number;
} {
  const version = useSyncExternalStore(
    subscribeProbeStore,
    getProbeStoreVersion,
    getProbeStoreVersion,
  );

  return { buffer: getProbeBuffer(), version };
}

function summarize(results: ProbeResult[]): LiveTargetState {
  if (results.length === 0) return EMPTY_STATE;
  let successes = 0;
  const successfulRTTs: number[] = [];
  for (const r of results) {
    if (r.success) {
      successes++;
      if (r.rtt_ms != null) successfulRTTs.push(r.rtt_ms);
    }
  }
  const successPct = (successes / results.length) * 100;
  const avgRTT =
    successfulRTTs.length > 0
      ? successfulRTTs.reduce((a, b) => a + b, 0) / successfulRTTs.length
      : null;
  return {
    results,
    successfulRTTs,
    successPct,
    avgRTTMs: avgRTT,
    latest: results[results.length - 1],
  };
}
