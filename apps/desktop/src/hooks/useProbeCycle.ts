import { useEffect, useState } from 'react';
import { onSidecarEvent } from '../lib/events';
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
 * Subscribes to `probe:cycle` events from the sidecar. Returns the most
 * recent cycle plus a counter of cycles received since mount (handy for
 * showing "alive" pulses in the UI).
 */
export function useProbeCycle() {
  const [latest, setLatest] = useState<ProbeCycleEvent | null>(null);
  const [count, setCount] = useState(0);

  useEffect(() => {
    const unlisten = onSidecarEvent<ProbeCycleEvent>('probe:cycle', (data) => {
      setLatest(data);
      setCount((c) => c + 1);
    });
    return () => {
      unlisten.then((fn) => fn()).catch(() => {});
    };
  }, []);

  return { latest, count };
}
