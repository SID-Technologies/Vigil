// Singleton store for live probe data.
//
// Why this exists: every page-level hook that subscribed to probe:cycle used
// component-local React state (useState / useRef). Navigating away from the
// Dashboard unmounted the hook, which torpedoed the cycle counter, the
// "latest cycle" snapshot, and the 5-minute rolling RTT buffer that powers
// the per-target sparklines. Coming back to the Dashboard then started over
// at "1 cycle, no history" — which made the app feel amnesiac.
//
// Fix: hold all of that state at module scope, subscribe to the IPC stream
// exactly once (on first reader), and never tear down. Hooks become thin
// wrappers that read the current snapshot via useSyncExternalStore. The
// store's lifetime is now the app's lifetime, not any one component's.
//
// `version` is the snapshot identity for useSyncExternalStore. Bumping it on
// every cycle is enough to retrigger renders; consumers then read the live
// state from the accessor functions below.

import { onSidecarEvent } from './events';
import type { ProbeCycleEvent, ProbeResult } from '../hooks/useProbeCycle';

/** Rolling-buffer window for live samples. 5 min ≈ 120 probes per target at default cadence. */
export const LIVE_WINDOW_MS = 5 * 60 * 1000;

let version = 0;
let count = 0;
let latest: ProbeCycleEvent | null = null;
const buffer = new Map<string, ProbeResult[]>();
const listeners = new Set<() => void>();

let unsubscribe: (() => void) | null = null;
let subscribePending = false;

function ensureSubscribed(): void {
  if (unsubscribe || subscribePending) return;
  subscribePending = true;
  // onSidecarEvent returns a Promise<unsubscribe>. We don't expose teardown —
  // the store lives for the lifetime of the app process.
  onSidecarEvent<ProbeCycleEvent>('probe:cycle', (event) => {
    count += 1;
    latest = event;

    const cutoff = Date.now() - LIVE_WINDOW_MS;
    for (const r of event.results) {
      const arr = buffer.get(r.target.label) ?? [];
      arr.push(r);
      while (arr.length > 0 && arr[0].ts_unix_ms < cutoff) {
        arr.shift();
      }
      buffer.set(r.target.label, arr);
    }

    version += 1;
    for (const l of listeners) l();
  })
    .then((unlisten) => {
      unsubscribe = unlisten;
      subscribePending = false;
    })
    .catch(() => {
      // If the IPC subscription fails on startup we leave subscribePending
      // false so a later reader can retry.
      subscribePending = false;
    });
}

/**
 * Subscribe to changes. Call site is `useSyncExternalStore`. The first
 * call lazily wires up the IPC listener.
 */
export function subscribeProbeStore(listener: () => void): () => void {
  ensureSubscribed();
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

/** Snapshot identity — primitive so React's === check is cheap and stable. */
export function getProbeStoreVersion(): number {
  return version;
}

export function getProbeCount(): number {
  return count;
}

export function getLatestCycle(): ProbeCycleEvent | null {
  return latest;
}

export function getProbeBuffer(): Map<string, ProbeResult[]> {
  return buffer;
}
