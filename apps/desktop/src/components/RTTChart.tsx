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
import { RefreshIndicator } from './RefreshIndicator';
import { ChartSkeleton } from './Skeleton';
import { useColorPalette } from '../hooks/useColorPalette';
import { useLiveSamples } from '../hooks/useLiveSamples';
import { useSamplesQuery } from '../hooks/useSamplesQuery';
import { useAllProbeTargets } from '../hooks/useAllProbeTargets';
import { fillRawGaps, generateTimeTicks } from '../lib/chartTime';
import type { RawSample } from '../lib/ipc';

interface RTTChartProps {
  selectedLabels: string[];
  onSetAll: (labels: string[]) => void;
  onClear: () => void;
}

/**
 * RTT trend over the last hour.
 *
 *   - 0 selected → median p50 across all targets (current/baseline view).
 *   - 1+ selected → one colored line per selected target.
 *
 * Median is the right default because the "is the network ok" question is
 * what the dashboard answers. Per-target lines is the comparative drill-in.
 *
 * Live indicators in the card header (pulsing dot + "updated Xs ago" pill)
 * keep the page feeling alive between the 30s data refetches.
 */
export function RTTChart({ selectedLabels, onSetAll, onClear }: RTTChartProps) {
  // Dashboard is the "is the network ok right now" surface — 15 min at
  // raw 2.5s gives ~360 points per target, so a 30-second blip is
  // visible at the moment it happens. The History page covers longer
  // windows. Recompute on every render so the window slides forward
  // with the wall clock; the React Query cache key handles dedup.
  const fromMs = Date.now() - 15 * 60 * 1000;
  const toMs = Date.now();

  const { tick } = useLiveSamples(); // for the live pulse heartbeat
  const allTargets = useAllProbeTargets();
  const samples = useSamplesQuery({
    fromMs,
    toMs,
    granularity: 'raw',
    targetLabels: selectedLabels.length > 0 ? selectedLabels : undefined,
  });
  const { getColor } = useColorPalette();

  const rows = samples.data && samples.data.granularity === 'raw' ? samples.data.rows : [];

  const data = useMemo(() => {
    if (selectedLabels.length === 0) {
      return fillRawGaps(rollupMedianRaw(rows));
    }
    return fillRawGaps(pivotRaw(rows, selectedLabels));
  }, [rows, selectedLabels]);

  const xTicks = useMemo(() => generateTimeTicks(fromMs, toMs, 5), [fromMs, toMs]);

  const isEmpty = data.length === 0;
  const isLoading = samples.isLoading && rows.length === 0;
  const isError = samples.isError;
  const lastUpdated = samples.dataUpdatedAt ? new Date(samples.dataUpdatedAt) : undefined;

  return (
    <Card
      title="RTT — last hour"
      trailing={
        <XStack gap="$2" alignItems="center">
          <PulsingDot color="var(--accentColor)" size={8} pulseKey={tick} />
          <Text fontSize={11} color="$color8">live</Text>
          <RefreshIndicator
            lastUpdated={lastUpdated}
            isFetching={samples.isFetching}
            onRefresh={() => samples.refetch()}
          />
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
      ) : isError ? (
        <ChartEmptyState
          headline="Couldn't fetch samples"
          detail="The sidecar may have stopped. Try restarting Vigil from the tray menu."
        />
      ) : isEmpty ? (
        <ChartEmptyState
          headline="Nothing yet"
          detail="Vigil's first 5-minute summary lands about 6 minutes after start. Until then, you'll see live data on the tiles below."
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

/**
 * Group raw probes by their (already-very-near) timestamp and take the
 * median RTT across reachable targets per cycle. Each `probe:cycle` event
 * fires every ~2.5s and contains one probe per target — the timestamps
 * within a cycle are within milliseconds of each other, so we round to
 * the nearest probe-cadence bucket so concurrent probes group cleanly.
 */
function rollupMedianRaw(rows: RawSample[]): MedianPoint[] {
  const byCycle = new Map<number, number[]>();
  for (const r of rows) {
    if (!r.success || r.rtt_ms == null) continue;
    // Round to the nearest second so a 13-target cycle's probes (which
    // span tens of ms) land in the same group.
    const t = Math.round(r.ts_unix_ms / 1000) * 1000;
    const arr = byCycle.get(t) ?? [];
    arr.push(r.rtt_ms);
    byCycle.set(t, arr);
  }
  return Array.from(byCycle.keys())
    .sort((a, b) => a - b)
    .map((t) => ({ ts: t, medianP50: median(byCycle.get(t)!) }));
}

function pivotRaw(rows: RawSample[], selectedLabels: string[]): PivotPoint[] {
  const allowed = new Set(selectedLabels);
  const byTs = new Map<number, PivotPoint>();
  for (const r of rows) {
    if (!allowed.has(r.target_label) || !r.success || r.rtt_ms == null) continue;
    const existing = byTs.get(r.ts_unix_ms) ?? { ts: r.ts_unix_ms };
    existing[r.target_label] = r.rtt_ms;
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
