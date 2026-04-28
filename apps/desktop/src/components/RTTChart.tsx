import { useMemo } from 'react';
import {
  Area,
  AreaChart,
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import { XStack, YStack, Text } from 'tamagui';

import { Card } from './Card';
import { ChartFilterBar } from './ChartFilterBar';
import { ChartTooltip } from './ChartTooltip';
import { PulsingDot } from './PulsingDot';
import { ChartSkeleton } from './Skeleton';
import { useColorPalette } from '../hooks/useColorPalette';
import { useLiveProbes } from '../hooks/useLiveSamples';
import { useAllProbeTargets } from '../hooks/useAllProbeTargets';
import type { ProbeResult } from '../hooks/useProbeCycle';
import { fillRawGaps, generateTimeTicks } from '../lib/chartTime';

interface RTTChartProps {
  selectedLabels: string[];
  onSetAll: (labels: string[]) => void;
  onClear: () => void;
}

/**
 * RTT trend for the dashboard. Reads directly from the in-memory probe
 * store (lib/probeStore) — no SQLite round-trip — so the chart updates
 * the moment a probe cycle arrives, matching the per-target tiles.
 *
 *   - 0 selected → median p50 across all targets (baseline view).
 *   - 1+ selected → one colored line per selected target.
 */
export function RTTChart({ selectedLabels, onSetAll, onClear }: RTTChartProps) {
  // 15-min sliding window. Recompute on every render so the X axis tracks
  // wall-clock; the buffer reference is stable so memoization still works.
  const fromMs = Date.now() - 15 * 60 * 1000;
  const toMs = Date.now();

  const { buffer, version } = useLiveProbes();
  const allTargets = useAllProbeTargets();
  const { getColor } = useColorPalette();

  // Flatten the per-target buffer to a single ProbeResult[] for the
  // existing rollup/pivot helpers. Filter to the active target set.
  const rows = useMemo(() => projectBuffer(buffer, selectedLabels, fromMs), [buffer, selectedLabels, fromMs, version]);

  const data = useMemo(() => {
    if (selectedLabels.length === 0) {
      return fillRawGaps(rollupMedianRaw(rows));
    }
    return fillRawGaps(pivotRaw(rows, selectedLabels));
  }, [rows, selectedLabels]);

  const xTicks = useMemo(() => generateTimeTicks(fromMs, toMs, 5), [fromMs, toMs]);

  const tick = version;
  const isEmpty = data.length === 0;
  const isLoading = buffer.size === 0;
  const isError = false;

  return (
    <Card
      title="RTT — last hour"
      trailing={
        <XStack gap="$2" alignItems="center">
          <PulsingDot color="var(--accentColor)" size={8} pulseKey={tick} />
          <Text fontSize={11} color="$color8">live</Text>
        </XStack>
      }
    >
      <ChartFilterBar
        allTargets={allTargets}
        selected={selectedLabels}
        onSetAll={onSetAll}
        onClear={onClear}
      />

      {isLoading ? (
        <ChartSkeleton />
      ) : isEmpty ? (
        <ChartEmptyState
          headline="Settling in"
          detail="The first probe cycle runs in just a couple of seconds — the chart fills as cycles arrive."
        />
      ) : selectedLabels.length === 0 ? (
        <MedianAreaChart
          data={data as MedianPoint[]}
          fromMs={fromMs}
          toMs={toMs}
          xTicks={xTicks}
        />
      ) : (
        <PerTargetLineChart
          data={data as PivotPoint[]}
          selectedLabels={selectedLabels}
          getColor={getColor}
          fromMs={fromMs}
          toMs={toMs}
          xTicks={xTicks}
        />
      )}

      {selectedLabels.length > 0 && (
        <XStack gap="$1.5" flexWrap="wrap">
          {selectedLabels.map((label) => (
            <XStack
              key={label}
              gap="$1.5"
              alignItems="center"
              paddingHorizontal="$1.5"
              paddingVertical="$1"
              borderRadius="$1"
              backgroundColor="$color3"
            >
              <YStack width={8} height={8} borderRadius={999} backgroundColor={getColor(label) as any} />
              <Text fontSize={11} color="$color11">{label}</Text>
            </XStack>
          ))}
        </XStack>
      )}
    </Card>
  );
}

// ============================================================================
// Sub-charts
// ============================================================================

interface MedianPoint {
  ts: number;
  medianP50: number;
}

function MedianAreaChart({
  data,
  fromMs,
  toMs,
  xTicks,
}: {
  data: MedianPoint[];
  fromMs: number;
  toMs: number;
  xTicks: number[];
}) {
  return (
    <YStack height={220}>
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={data} margin={{ top: 4, right: 8, bottom: 0, left: -16 }}>
          <defs>
            <linearGradient id="rttFill" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="var(--accentColor)" stopOpacity={0.35} />
              <stop offset="100%" stopColor="var(--accentColor)" stopOpacity={0.02} />
            </linearGradient>
          </defs>
          <CartesianGrid stroke="var(--borderColor)" strokeDasharray="3 3" />
          <XAxis
            dataKey="ts"
            type="number"
            scale="time"
            domain={[fromMs, toMs]}
            ticks={xTicks}
            tickFormatter={fmtTime}
            {...axisStyle}
          />
          <YAxis {...axisStyle} width={42} label={yLabel} />
          <Tooltip
            content={<ChartTooltip formatLabel={fmtTimeLong} unit="ms" caption="median p50 across targets" />}
            cursor={{ stroke: 'var(--color8)', strokeWidth: 1, strokeDasharray: '3 3' }}
          />
          <Area
            type="monotone"
            dataKey="medianP50"
            stroke="var(--accentColor)"
            strokeWidth={2}
            fill="url(#rttFill)"
            isAnimationActive={false}
            connectNulls={false}
          />
        </AreaChart>
      </ResponsiveContainer>
    </YStack>
  );
}

interface PivotPoint {
  ts: number;
  [targetLabel: string]: number | undefined;
}

function PerTargetLineChart({
  data,
  selectedLabels,
  getColor,
  fromMs,
  toMs,
  xTicks,
}: {
  data: PivotPoint[];
  selectedLabels: string[];
  getColor: (label: string) => string;
  fromMs: number;
  toMs: number;
  xTicks: number[];
}) {
  return (
    <YStack height={220}>
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={data} margin={{ top: 4, right: 8, bottom: 0, left: -16 }}>
          <CartesianGrid stroke="var(--borderColor)" strokeDasharray="3 3" />
          <XAxis
            dataKey="ts"
            type="number"
            scale="time"
            domain={[fromMs, toMs]}
            ticks={xTicks}
            tickFormatter={fmtTime}
            {...axisStyle}
          />
          <YAxis {...axisStyle} width={42} label={yLabel} />
          <Tooltip
            content={<ChartTooltip formatLabel={fmtTimeLong} unit="ms" />}
            cursor={{ stroke: 'var(--color8)', strokeWidth: 1, strokeDasharray: '3 3' }}
          />
          {selectedLabels.map((label) => (
            <Line
              key={label}
              type="monotone"
              dataKey={label}
              stroke={getColor(label)}
              strokeWidth={2}
              dot={false}
              isAnimationActive={false}
              connectNulls={false}
            />
          ))}
        </LineChart>
      </ResponsiveContainer>
    </YStack>
  );
}

function ChartEmptyState({ headline, detail }: { headline: string; detail: string }) {
  return (
    <YStack height={220} alignItems="center" justifyContent="center" gap="$1">
      <Text fontSize={13} color="$color10" fontWeight="600">
        {headline}
      </Text>
      <Text fontSize={11} color="$color8" textAlign="center" maxWidth={360}>
        {detail}
      </Text>
    </YStack>
  );
}

// ============================================================================
// Data shaping
// ============================================================================

// projectBuffer flattens the per-target store buffer to a time-window-
// scoped slice. The store mutates the buffer in place; callers depend on
// `version` for memo invalidation, not on buffer reference identity.
function projectBuffer(
  buffer: Map<string, ProbeResult[]>,
  selectedLabels: string[],
  fromMs: number,
): ProbeResult[] {
  const out: ProbeResult[] = [];
  const filterByLabel = selectedLabels.length > 0;
  const allowed = filterByLabel ? new Set(selectedLabels) : null;
  for (const [label, results] of buffer.entries()) {
    if (allowed && !allowed.has(label)) continue;
    for (const r of results) {
      if (r.ts_unix_ms >= fromMs) out.push(r);
    }
  }
  return out;
}

// Group probes by their (already-very-near) timestamp and take the median
// RTT across reachable targets per cycle. Each cycle's probes span tens of
// ms, so round to the nearest second to group them cleanly.
function rollupMedianRaw(rows: ProbeResult[]): MedianPoint[] {
  const byCycle = new Map<number, number[]>();
  for (const r of rows) {
    if (!r.success || r.rtt_ms == null) continue;
    const t = Math.round(r.ts_unix_ms / 1000) * 1000;
    const arr = byCycle.get(t) ?? [];
    arr.push(r.rtt_ms);
    byCycle.set(t, arr);
  }
  return Array.from(byCycle.keys())
    .sort((a, b) => a - b)
    .map((t) => ({ ts: t, medianP50: median(byCycle.get(t)!) }));
}

function pivotRaw(rows: ProbeResult[], selectedLabels: string[]): PivotPoint[] {
  const allowed = new Set(selectedLabels);
  const byTs = new Map<number, PivotPoint>();
  for (const r of rows) {
    if (!allowed.has(r.target.label) || !r.success || r.rtt_ms == null) continue;
    const existing = byTs.get(r.ts_unix_ms) ?? { ts: r.ts_unix_ms };
    existing[r.target.label] = r.rtt_ms;
    byTs.set(r.ts_unix_ms, existing);
  }
  return Array.from(byTs.values()).sort((a, b) => a.ts - b.ts);
}

function median(xs: number[]): number {
  const sorted = xs.slice().sort((a, b) => a - b);
  const mid = Math.floor(sorted.length / 2);
  return sorted.length % 2 === 0 ? (sorted[mid - 1] + sorted[mid]) / 2 : sorted[mid];
}

// ============================================================================
// Shared chart styling
// ============================================================================

const axisStyle = {
  tick: { fontSize: 10, fill: 'var(--color9)' },
  stroke: 'var(--borderColor)',
};

const yLabel = {
  value: 'ms',
  angle: -90,
  position: 'insideLeft' as const,
  fill: 'var(--color9)',
  fontSize: 10,
  offset: 16,
};

const tooltipStyle = {
  contentStyle: {
    background: 'var(--color2)',
    border: '1px solid var(--borderColor)',
    borderRadius: 6,
    fontSize: 11,
  },
};

function msFormatter(name: string) {
  return (v: number) => [`${v.toFixed(2)} ms`, name];
}

function fmtTime(ms: number): string {
  return new Date(ms).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
}

function fmtTimeLong(ms: number): string {
  return new Date(ms).toLocaleString([], {
    hour: '2-digit',
    minute: '2-digit',
    month: 'short',
    day: 'numeric',
  });
}
