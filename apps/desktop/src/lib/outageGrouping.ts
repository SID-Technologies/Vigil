// Outage grouping for the Outages list view.
//
// The detector records one outage per probe scope. When a service has
// multiple probes (e.g., outlook_icmp + outlook_tcp443) and both fail at
// once, that's logically one incident — but the DB sees two rows. This
// module folds them into a single user-visible incident.
//
// Grouping criteria:
//   1. Same service name. Default targets follow `<service>_<kind>` where
//      kind is `icmp`, `tcp<port>`, or `udp`. Strip the suffix to get the
//      service. Custom targets without a recognized suffix become their
//      own single-probe service (full label preserved).
//   2. Overlapping time windows. Ongoing outages extend to `nowMs` for
//      the overlap test.
//
// Per-probe rows stay intact in the DB — this is a view-layer collapse,
// so CSV exports and reports retain full granularity.

import type { Outage } from '../hooks/useOutages';

const PROBE_KIND_SUFFIX = /_(icmp|tcp\d*|udp)$/;

/**
 * Strip the probe-kind suffix from a scope to get the service name. The
 * `network` scope passes through unchanged so the grouping logic can treat
 * network-wide outages as their own bucket.
 */
export function serviceName(scope: string): string {
  if (scope === 'network') return 'network';
  const label = scope.startsWith('target:') ? scope.slice('target:'.length) : scope;
  return label.replace(PROBE_KIND_SUFFIX, '');
}

export interface OutageGroup {
  /** Stable key for React. */
  key: string;
  /** Derived service name, e.g. "outlook" or "cloudflare_dns". */
  service: string;
  /** Drives icon/styling — network outages render with a distinct accent. */
  kind: 'network' | 'service';
  /** Earliest member start. */
  startMs: number;
  /** Latest member end, or null if any member is still ongoing. */
  endMs: number | null;
  /** Source outage rows that compose this incident. */
  members: Outage[];
  /** Distinct probe labels (e.g., ["outlook_icmp", "outlook_tcp443"]). */
  probeLabels: string[];
}

/**
 * Bucket outages by service name and merge those whose time windows touch
 * into single incidents.
 *
 * `nowMs` defaults to the current time; passed in for testability and so
 * ongoing outages anchor to the same instant across the run.
 */
export function groupOutages(all: Outage[], nowMs: number = Date.now()): OutageGroup[] {
  if (all.length === 0) return [];

  const byService = new Map<string, Outage[]>();
  for (const o of all) {
    const s = serviceName(o.scope);
    const arr = byService.get(s) ?? [];
    arr.push(o);
    byService.set(s, arr);
  }

  const groups: OutageGroup[] = [];

  for (const [service, members] of byService) {
    members.sort((a, b) => a.start_ts_unix_ms - b.start_ts_unix_ms);

    let current: Outage[] = [];
    let currentEndMs = -Infinity;

    const flush = () => {
      if (current.length === 0) return;
      const startMs = current[0].start_ts_unix_ms;
      let endMs: number | null = -Infinity;
      for (const m of current) {
        if (m.end_ts_unix_ms == null) {
          endMs = null;
          break;
        }
        if (m.end_ts_unix_ms > (endMs as number)) endMs = m.end_ts_unix_ms;
      }
      const probeLabels = Array.from(
        new Set(
          current.map((m) =>
            m.scope === 'network' ? 'network' : m.scope.replace('target:', ''),
          ),
        ),
      );
      groups.push({
        key: `${service}-${startMs}`,
        service,
        kind: service === 'network' ? 'network' : 'service',
        startMs,
        endMs,
        members: current.slice(),
        probeLabels,
      });
      current = [];
      currentEndMs = -Infinity;
    };

    for (const m of members) {
      const mEndMs = m.end_ts_unix_ms ?? nowMs;
      if (current.length === 0) {
        current.push(m);
        currentEndMs = mEndMs;
        continue;
      }
      if (m.start_ts_unix_ms <= currentEndMs) {
        current.push(m);
        if (mEndMs > currentEndMs) currentEndMs = mEndMs;
        continue;
      }
      flush();
      current.push(m);
      currentEndMs = mEndMs;
    }
    flush();
  }

  // Newest first — matches the user's likely scan order in the list.
  groups.sort((a, b) => b.startMs - a.startMs);
  return groups;
}
