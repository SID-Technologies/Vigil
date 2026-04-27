import { useEffect, useRef, useState } from 'react';
import { onSidecarEvent } from '../lib/events';
import type { ProbeCycleEvent, ProbeResult } from './useProbeCycle';

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
 * Maintains a rolling window of `windowMs` per target, fed by `probe:cycle`
 * events. Each event contains every target's latest probe, so we update all
 * 13 buffers in one pass and trim by timestamp afterward.
 *
 * Buffer lives in a ref to avoid spawning new Map identities on every cycle.
 * State is just a tick counter that increments after each update — components
 * memo on tick so renders stay cheap.
 *
 * Default window: 5 minutes. Steady-state cycle rate ~24/min × 5 min = ~120
 * probes per target retained, ~1.5KB/target × 13 targets = ~20KB total.
 */
export function useLiveSamples(windowMs = 5 * 60 * 1000): {
  states: Map<string, LiveTargetState>;
  /** Increments on each probe:cycle — handy as a render trigger / pulse counter. */
  tick: number;
} {
  const bufferRef = useRef<Map<string, ProbeResult[]>>(new Map());
  const [tick, setTick] = useState(0);

  useEffect(() => {
    let active = true;
    const unlistenPromise = onSidecarEvent<ProbeCycleEvent>('probe:cycle', (event) => {
      if (!active) return;
      const cutoff = Date.now() - windowMs;
      for (const r of event.results) {
        const arr = bufferRef.current.get(r.target.label) ?? [];
        arr.push(r);
        // Trim from the front while older than the window.
        while (arr.length > 0 && arr[0].ts_unix_ms < cutoff) {
          arr.shift();
        }
        bufferRef.current.set(r.target.label, arr);
      }
      setTick((t) => t + 1);
    });
    return () => {
      active = false;
      unlistenPromise.then((fn) => fn()).catch(() => {});
    };
  }, [windowMs]);

  const states = computeStates(bufferRef.current);
  return { states, tick };
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
