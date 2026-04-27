import { useMemo } from 'react';

import { useProbeCycle } from './useProbeCycle';
import { useTargets } from './useTargets';
import type { ProbeKind, Target } from '../lib/ipc';

/**
 * Returns the union of DB-persisted targets and ephemeral targets observed
 * in live probe cycles.
 *
 * Background: the dynamic gateway probe (`router_icmp`) is added by the
 * Go monitor at startup based on the detected default gateway. It runs
 * every cycle and persists samples like every other probe — but it's
 * never written to the `targets` table because the gateway can change
 * (different network, DHCP renewal, VPN, etc.). Without this hook, target
 * filters in the UI miss it entirely: it shows up in the live grid but
 * not in any selector.
 *
 * Strategy: take the targets table (`useTargets`) as the canonical list,
 * then walk the most recent probe cycle and add any label that isn't
 * already there as a synthetic Target. The synthetic entry uses an
 * `ephemeral:` ID prefix so callers can detect it if needed (e.g. to
 * disable Edit/Delete actions in the Targets page).
 */
export function useAllProbeTargets(): Target[] {
  const targets = useTargets();
  const { latest } = useProbeCycle();

  return useMemo(() => {
    const dbRows = targets.data ?? [];
    const known = new Set(dbRows.map((t) => t.label));
    const out: Target[] = [...dbRows];

    if (latest?.results) {
      for (const r of latest.results) {
        if (!known.has(r.target.label)) {
          out.push({
            id: `ephemeral:${r.target.label}`,
            label: r.target.label,
            kind: r.target.kind as ProbeKind,
            host: r.target.host,
            port: r.target.port,
            enabled: true,
            // Treat ephemerals as builtin so per-kind filter buttons
            // include them and they appear sticky in the UI.
            is_builtin: true,
          });
          known.add(r.target.label);
        }
      }
    }
    return out;
  }, [targets.data, latest]);
}
