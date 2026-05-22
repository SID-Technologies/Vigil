// Outage grouping for the Outages list view.
//
// The detector records one outage per probe scope. Two collapse passes
// turn that raw stream into incidents users can scan:
//
//   1. Same-service merge — outlook_icmp + outlook_tcp443 → one "outlook"
//      group (service name parsed by stripping `_icmp`/`_tcp<port>`/`_udp`
//      suffixes; custom targets without a recognizable suffix become
//      their own single-probe "service").
//
//   2. Burst merge — service groups whose starts are within
//      BURST_WINDOW_MS AND whose intervals overlap fold into one
//      multi-service incident. Catches the "google_stun + cloudflare_stun
//      both fail in the same minute" case: simultaneous cross-provider
//      failures almost always share a root cause (UDP blocked on the
//      network, DNS recursion broken, etc.).
//
// Critically, burst merge only kicks in when starts are clustered, not
// just when intervals overlap. A permanent ICMP-blocked router outage
// (ongoing for hours) does NOT swallow every unrelated outage that opens
// during its lifetime — those start far later than the cluster anchor.
//
// Per-probe rows stay intact in the DB — this is a view-layer collapse,
// so CSV exports and reports retain full granularity.

import type { Outage } from '../hooks/useOutages';

const PROBE_KIND_SUFFIX = /_(icmp|tcp\d*|udp)$/;

/** How tight start times must cluster for a burst merge. 60s = "in the same minute". */
export const BURST_WINDOW_MS = 60_000;

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
  /**
   * Distinct service names contributing to this incident. Length 1 for a
   * normal single-service group; length ≥2 for a burst-merged incident.
   */
  services: string[];
  /**
   * Drives icon/styling — `network` if any member's scope is `network`,
   * otherwise `service`.
   */
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
        services: [service],
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

  // Burst-merge pass: collapse service groups whose starts are clustered
  // (within BURST_WINDOW_MS) AND whose intervals overlap. Captures
  // cross-provider correlated failures (the "google_stun + cloudflare_stun
  // both fail at 14:17" case).
  const merged = mergeBursts(groups, nowMs);

  // Newest first — matches the user's likely scan order in the list.
  merged.sort((a, b) => b.startMs - a.startMs);
  return merged;
}

function mergeBursts(groups: OutageGroup[], nowMs: number): OutageGroup[] {
  if (groups.length <= 1) return groups;

  const byStart = groups.slice().sort((a, b) => a.startMs - b.startMs);
  const out: OutageGroup[] = [];

  let cluster: OutageGroup[] = [];
  let anchorStartMs = -Infinity;
  let clusterEndMs = -Infinity;

  const flush = () => {
    if (cluster.length === 0) return;
    if (cluster.length === 1) {
      out.push(cluster[0]);
      cluster = [];
      return;
    }

    out.push(mergeClusterMembers(cluster));
    cluster = [];
  };

  for (const g of byStart) {
    const gEndMs = g.endMs ?? nowMs;

    if (cluster.length === 0) {
      cluster.push(g);
      anchorStartMs = g.startMs;
      clusterEndMs = gEndMs;
      continue;
    }

    const startCloseEnough = g.startMs - anchorStartMs <= BURST_WINDOW_MS;
    const intervalsOverlap = g.startMs <= clusterEndMs;

    if (startCloseEnough && intervalsOverlap) {
      cluster.push(g);
      if (gEndMs > clusterEndMs) clusterEndMs = gEndMs;
      continue;
    }

    flush();
    cluster.push(g);
    anchorStartMs = g.startMs;
    clusterEndMs = gEndMs;
  }

  flush();
  return out;
}

function mergeClusterMembers(cluster: OutageGroup[]): OutageGroup {
  const services = Array.from(new Set(cluster.flatMap((g) => g.services))).sort();
  const probeLabels = Array.from(new Set(cluster.flatMap((g) => g.probeLabels))).sort();
  const members = cluster.flatMap((g) => g.members);
  const startMs = Math.min(...cluster.map((g) => g.startMs));

  let endMs: number | null = -Infinity;
  for (const g of cluster) {
    if (g.endMs == null) {
      endMs = null;
      break;
    }
    if (g.endMs > (endMs as number)) endMs = g.endMs;
  }

  const isNetwork = cluster.some((g) => g.kind === 'network');

  return {
    key: `burst-${services.join('+')}-${startMs}`,
    services,
    kind: isNetwork ? 'network' : 'service',
    startMs,
    endMs,
    members,
    probeLabels,
  };
}
