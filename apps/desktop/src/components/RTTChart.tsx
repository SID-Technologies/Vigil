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
import type { AggregatedRow } from '../lib/ipc';

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
  const fromMs = useMemo(() => Date.now() - 60 * 60 * 1000, []);
  const toMs = useMemo(() => Date.now(), []);

  const { tick } = useLiveSamples(); // for the live pulse heartbeat
  const allTargets = useAllProbeTargets();
  const samples = useSamplesQuery({
    fromMs,
    toMs,
    granularity: '5min',
    targetLabels: selectedLabels.length > 0 ? selectedLabels : undefined,
  });
  const { getColor } = useColorPalette();

  const rows = samples.data && samples.data.granularity === '5min' ? samples.data.rows : [];

  const data = useMemo(() => {
    if (selectedLabels.length === 0) {
      return rollupMedian(rows);
    }
    return pivotByTarget(rows, selectedLabels);
  }, [rows, selectedLabels]);

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
        <MedianAreaChart data={data as MedianPoint[]} />
      ) : (
        <PerTargetLineChart
          data={data as PivotPoint[]}
          selectedLabels={selectedLabels}
          getColor={getColor}
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

function MedianAreaChart({ data }: { data: MedianPoint[] }) {
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
          <XAxis dataKey="ts" tickFormatter={fmtTime} {...axisStyle} />
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
}: {
  data: PivotPoint[];
  selectedLabels: string[];
  getColor: (label: string) => string;
}) {
  return (
    <YStack height={220}>
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={data} margin={{ top: 4, right: 8, bottom: 0, left: -16 }}>
          <CartesianGrid stroke="var(--borderColor)" strokeDasharray="3 3" />
          <XAxis dataKey="ts" tickFormatter={fmtTime} {...axisStyle} />
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
              connectNulls
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

function rollupMedian(rows: AggregatedRow[]): MedianPoint[] {
  const byBucket = new Map<number, number[]>();
  for (const r of rows) {
    if (r.rtt_p50_ms == null) continue;
    const arr = byBucket.get(r.bucket_start_unix_ms) ?? [];
    arr.push(r.rtt_p50_ms);
    byBucket.set(r.bucket_start_unix_ms, arr);
  }
  return Array.from(byBucket.keys())
    .sort((a, b) => a - b)
    .map((b) => ({ ts: b, medianP50: median(byBucket.get(b)!) }));
}

function pivotByTarget(rows: AggregatedRow[], selectedLabels: string[]): PivotPoint[] {
  const allowed = new Set(selectedLabels);
  const byTs = new Map<number, PivotPoint>();
  for (const r of rows) {
    if (!allowed.has(r.target_label) || r.rtt_p50_ms == null) continue;
    const existing = byTs.get(r.bucket_start_unix_ms) ?? { ts: r.bucket_start_unix_ms };
    existing[r.target_label] = r.rtt_p50_ms;
    byTs.set(r.bucket_start_unix_ms, existing);
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
