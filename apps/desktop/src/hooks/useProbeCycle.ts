import { useSyncExternalStore } from 'react';
import {
  getLatestCycle,
  getProbeCount,
  getProbeStoreVersion,
  subscribeProbeStore,
} from '../lib/probeStore';
import type { ProbeKind } from '../lib/ipc';

export interface ProbeResult {
  ts_unix_ms: number;
  target: {
    label: string;
    kind: ProbeKind;
    host: string;
    port?: number;
  };
  success: boolean;
  rtt_ms?: number;
  error?: string;
}

export interface ProbeCycleEvent {
  ts_unix_ms: number;
  total: number;
  ok: number;
  fail: number;
  results: ProbeResult[];
}

/**
 * Reads the most recent probe cycle and the cycle counter from the singleton
 * probe store. Both values persist across route changes — the store is owned
 * at module scope, not by any one component.
 *
 * The hook subscribes to a primitive `version` number; React rerenders only
 * when it changes, and the live data is read fresh from the store each time.
 */
export function useProbeCycle() {
  useSyncExternalStore(subscribeProbeStore, getProbeStoreVersion, getProbeStoreVersion);
  return {
    latest: getLatestCycle(),
    count: getProbeCount(),
  };
}
