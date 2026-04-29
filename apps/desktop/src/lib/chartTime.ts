// Helpers for time-scaled charts.
//
// The dashboard and History charts both render time-series RTT lines. Two
// problems they share:
//
//   1. recharts defaults to a CATEGORICAL X axis when type isn't set —
//      every point gets equal width, so a 4-hour stretch with no data
//      collapses to "two adjacent points." The chart effectively lies
//      about when things happened.
//
//   2. Even with a numeric/time axis, recharts will draw a straight line
//      from the last point before a gap to the first point after it, which
//      visually claims "RTT was steady at X ms across that window" — also
//      a lie.
//
// These helpers fix both: synthesize null entries at every missing bucket
// (so connectNulls={false} produces a visible break in the line) and pick
// sensible numeric tick positions for the X axis.

/** Bucket spacing for each aggregation granularity. */
export const BUCKET_INTERVAL_MS = {
  '1min': 60 * 1000,
  '5min': 5 * 60 * 1000,
  '1h': 60 * 60 * 1000,
} as const;

// CycleBucketMs groups concurrent probes from the same cycle. The 13 probes
// in one cycle finish within ~10–50ms of each other; bucketing at 100ms
// reliably merges them into one X position. Coarser bucketing (e.g. 1s) is
// wrong because cycle intervals like 2.5s snap unevenly, producing visible
// 2s/3s sawtooth gaps along the X axis.
export const CycleBucketMs = 100;

// CycleBucket rounds a probe timestamp to the nearest cycle bucket so the
// 13 concurrent probes in one cycle land on the same X position.
export function CycleBucket(ts: number): number {
  return Math.round(ts / CycleBucketMs) * CycleBucketMs;
}

export type BucketGranularity = keyof typeof BUCKET_INTERVAL_MS;

interface TimePoint {
  ts: number;
  [seriesKey: string]: number | undefined | null;
}

/**
 * Insert empty placeholder rows at every expected bucket timestamp where
 * data is missing. recharts treats `undefined` series values as null, so
 * with `connectNulls={false}` the line breaks across each missing bucket
 * — making real gaps visible at their real positions on a numeric X axis.
 */
export function fillBucketGaps<T extends TimePoint>(
  points: T[],
  fromMs: number,
  toMs: number,
  intervalMs: number,
): T[] {
  const present = new Map<number, T>();
  for (const p of points) present.set(p.ts, p);

  // Snap window to bucket grid so we don't show partial-bucket positions.
  const start = Math.floor(fromMs / intervalMs) * intervalMs;
  const end = Math.ceil(toMs / intervalMs) * intervalMs;

  const out: T[] = [];
  for (let t = start; t <= end; t += intervalMs) {
    const exact = present.get(t);
    if (exact) {
      out.push(exact);
    } else {
      // Empty placeholder — every series key resolves to undefined, which
      // recharts skips when connectNulls=false.
      out.push({ ts: t } as T);
    }
  }
  return out;
}

/**
 * For raw (un-bucketed) samples, detect anomalous gaps by comparing
 * consecutive timestamps to a threshold and insert a single null marker
 * mid-gap. The threshold defaults to 30 seconds — comfortably above the
 * 2.5-sec default probe interval but small enough to catch any real
 * outage or pause in monitoring.
 */
export function fillRawGaps<T extends TimePoint>(
  points: T[],
  thresholdMs = 30_000,
): T[] {
  if (points.length < 2) return points;
  const out: T[] = [points[0]];
  for (let i = 1; i < points.length; i++) {
    const gap = points[i].ts - points[i - 1].ts;
    if (gap > thresholdMs) {
      // Single null marker in the middle of the gap is enough to break
      // the line — adding more would just slow rendering.
      out.push({ ts: points[i - 1].ts + Math.floor(gap / 2) } as T);
    }
    out.push(points[i]);
  }
  return out;
}

/**
 * Return a sensible round-number interval for `n` ticks across `spanMs`.
 * Snaps to whole minutes / hours / days so labels stay legible.
 */
export function niceTimeInterval(spanMs: number, targetCount = 6): number {
  const step = spanMs / targetCount;
  const intervals = [
    60_000,            // 1m
    2 * 60_000,        // 2m
    5 * 60_000,        // 5m
    10 * 60_000,       // 10m
    15 * 60_000,       // 15m
    30 * 60_000,       // 30m
    3_600_000,         // 1h
    2 * 3_600_000,     // 2h
    3 * 3_600_000,     // 3h
    6 * 3_600_000,     // 6h
    12 * 3_600_000,    // 12h
    86_400_000,        // 1d
    2 * 86_400_000,    // 2d
    7 * 86_400_000,    // 1w
  ];
  for (const i of intervals) {
    if (i >= step) return i;
  }
  return intervals[intervals.length - 1];
}

/** Generate axis tick positions that snap to nice round times. */
export function generateTimeTicks(
  fromMs: number,
  toMs: number,
  targetCount = 6,
): number[] {
  const interval = niceTimeInterval(toMs - fromMs, targetCount);
  const start = Math.ceil(fromMs / interval) * interval;
  const ticks: number[] = [];
  for (let t = start; t <= toMs; t += interval) ticks.push(t);
  return ticks;
}
