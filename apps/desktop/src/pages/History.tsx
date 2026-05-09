import { useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { Export } from '@phosphor-icons/react';

import { useAccent } from '@repo/configs/themeController';
import {
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import { Button, XStack, YStack, Text } from 'tamagui';

import { Card } from '../components/Card';
import { ChartTooltip } from '../components/ChartTooltip';
import { GenerateReportModal } from '../components/GenerateReportModal';
import { InfoLabel } from '../components/InfoLabel';
import { PageHeader } from '../components/PageHeader';
import { ChartSkeleton, TableRowSkeleton } from '../components/Skeleton';
import { TargetMultiSelect } from '../components/TargetMultiSelect';
import { TimeRangePicker, defaultRange, type TimeRange } from '../components/TimeRangePicker';
import { useColorPalette } from '../hooks/useColorPalette';
import { useSamplesQuery } from '../hooks/useSamplesQuery';
import { useAllProbeTargets } from '../hooks/useAllProbeTargets';
import {
  BUCKET_INTERVAL_MS,
  CycleBucket,
  fillBucketGaps,
  fillRawGaps,
  generateTimeTicks,
} from '../lib/chartTime';
import type { AggregatedRow, RawSample } from '../lib/ipc';

/**
 * History — full time-range explorer.
 *
 * URL query params (so deep links from the Dashboard's per-target ↗ icons
 * work and the page can be bookmarked / shared):
 *
 *   ?target=foo               preselect a single target
 *   ?target=foo,bar           preselect multiple
 *   ?from=<ms>&to=<ms>        explicit window (used for custom ranges)
 *   ?range=<ms>               legacy preset duration; kept for back-compat
 *
 * Granularity is auto-resolved by the sidecar: ≤2h raw, ≤7d 5min, >7d 1h.
 * The page transparently handles both raw and aggregated row shapes.
 */
type MetricKey = 'p50' | 'p95' | 'p99' | 'max';

const METRICS: { key: MetricKey; label: string; explain: string }[] = [
  { key: 'p50', label: 'P50', explain: 'Median per bucket — the typical probe.' },
  { key: 'p95', label: 'P95', explain: 'Slow tail — what the worst 5% of probes hit.' },
  { key: 'p99', label: 'P99', explain: 'Top 1% — catches frequent spikes.' },
  { key: 'max', label: 'Max', explain: 'Worst single probe per bucket — rare incidents.' },
];

const AGG_FIELD: Record<MetricKey, keyof AggregatedRow> = {
  p50: 'rtt_p50_ms',
  p95: 'rtt_p95_ms',
  p99: 'rtt_p99_ms',
  max: 'rtt_max_ms',
};

export function HistoryPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const allTargets = useAllProbeTargets();
  const { getColor } = useColorPalette();
  const accent = useAccent();

  // Initialize from URL once. Keep selection + range in component state,
  // then sync to URL on change so back/forward navigation works.
  const [selected, setSelected] = useState<string[]>(() => parseTargets(searchParams.get('target')));
  const [range, setRange] = useState<TimeRange>(() => parseRangeFromUrl(searchParams) ?? defaultRange());
  const [metric, setMetric] = useState<MetricKey>('p95');
  // Honor ?report=1 from the app menu's "New Report" command — opens the
  // modal automatically. The param is then stripped from the URL so a
  // back-button repeat doesn't re-trigger.
  const [reportOpen, setReportOpen] = useState(() => searchParams.get('report') === '1');

  useEffect(() => {
    if (searchParams.get('report') === '1') {
      const next = new URLSearchParams(searchParams);
      next.delete('report');
      setSearchParams(next, { replace: true });
    }
    // Mount-only.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const { fromMs, toMs } = range;
  const rangeMs = toMs - fromMs;

  // Sync URL when selection or range changes.
  useEffect(() => {
    const next = new URLSearchParams(searchParams);
    if (selected.length > 0) next.set('target', selected.join(','));
    else next.delete('target');
    next.set('from', String(fromMs));
    next.set('to', String(toMs));
    next.delete('range'); // supersede legacy param
    if (next.toString() !== searchParams.toString()) {
      setSearchParams(next, { replace: true });
    }
  }, [selected, fromMs, toMs, searchParams, setSearchParams]);

  const samples = useSamplesQuery({
    fromMs,
    toMs,
    targetLabels: selected.length > 0 ? selected : undefined,
    refetchIntervalMs: 60_000,
  });

  const targetsToChart = selected.length > 0 ? selected : allTargets.map((t) => t.label);

  return (
    <YStack flex={1}>
      <PageHeader
        title="History"
        blurb="Time-range explorer — pick a window and one or more targets, see RTT trends and uptime."
        trailing={
          <Button
            size="$3"
            backgroundColor="$accentBackground"
            color="$accentColor"
            icon={<Export size={14} color={accent} />}
            onPress={() => setReportOpen(true)}
            hoverStyle={{ opacity: 0.9 }}
          >
            Generate report
          </Button>
        }
      />
      <GenerateReportModal
        open={reportOpen}
        onOpenChange={setReportOpen}
        fromMs={fromMs}
        toMs={toMs}
        targets={selected}
        windowLabel={humanizeRange(rangeMs)}
      />

      <YStack padding="$4" gap="$4" maxWidth={1300} width="100%" alignSelf="center">
        <Card variant="quiet">
          <YStack gap="$3">
            <XStack gap="$3" alignItems="center" flexWrap="wrap">
              <Text fontSize={11} color="$color10" letterSpacing={0.4} fontWeight="600">
                WINDOW
              </Text>
              <TimeRangePicker value={range} onChange={setRange} />
              <Text fontSize={11} color="$color8" marginLeft="$2">
                granularity: {predictedGranularity(rangeMs)}
              </Text>
            </XStack>
            <YStack height={1} backgroundColor="$borderColor" />
            <YStack gap="$2">
              <Text fontSize={11} color="$color10" letterSpacing={0.4} fontWeight="600">
                TARGETS
              </Text>
              <TargetMultiSelect
                allTargets={allTargets}
                selected={selected}
                onToggle={(label) =>
                  setSelected((prev) =>
                    prev.includes(label) ? prev.filter((l) => l !== label) : [...prev, label],
                  )
                }
                onSetAll={setSelected}
                onClear={() => setSelected([])}
                emptyMessage="All targets shown — click one to focus."
              />
            </YStack>
          </YStack>
        </Card>

        <Card
          title={`RTT — ${humanizeRange(rangeMs)}`}
          trailing={
            samples.data && samples.data.granularity !== 'raw' ? (
              <XStack gap="$1" alignItems="center">
                {METRICS.map((m) => (
                  <MetricChip
                    key={m.key}
                    label={m.label}
                    explain={m.explain}
                    active={metric === m.key}
                    onPress={() => setMetric(m.key)}
                  />
                ))}
              </XStack>
            ) : null
          }
        >
          {samples.isLoading && !samples.data ? (
            <ChartSkeleton height={260} />
          ) : samples.isError ? (
            <ChartEmpty
              headline="Couldn't fetch samples"
              detail="The sidecar may have stopped. Try restarting Vigil from the tray menu."
            />
          ) : !samples.data || samples.data.rows.length === 0 ? (
            <ChartEmpty
              headline="Nothing recorded for this stretch"
              detail="Either Vigil hasn't been running long enough, or every probe was disabled. Try a longer window or check the Targets page."
            />
          ) : (
            <HistoryChart
              data={samples.data}
              targetsToChart={targetsToChart}
              getColor={getColor}
              fromMs={fromMs}
              toMs={toMs}
              metric={metric}
            />
          )}
        </Card>

        {samples.data ? (
          <SummaryStats data={samples.data} targetsToChart={targetsToChart} />
        ) : samples.isLoading ? (
          <Card title="By target">
            <YStack gap="$1">
              <TableRowSkeleton columns={6} />
              <TableRowSkeleton columns={6} />
              <TableRowSkeleton columns={6} />
              <TableRowSkeleton columns={6} />
            </YStack>
          </Card>
        ) : null}
      </YStack>
    </YStack>
  );
}

// ============================================================================
// Chart (handles both raw and aggregated shapes)
// ============================================================================

interface PivotPoint {
  ts: number;
  [label: string]: number | undefined;
}

function HistoryChart({
  data,
  targetsToChart,
  getColor,
  fromMs,
  toMs,
  metric,
}: {
  data: NonNullable<ReturnType<typeof useSamplesQuery>['data']>;
  targetsToChart: string[];
  getColor: (label: string) => string;
  fromMs: number;
  toMs: number;
  metric: MetricKey;
}) {
  const points = useMemo<PivotPoint[]>(() => {
    if (data.granularity === 'raw') {
      // Raw probes don't have a fixed bucket cadence — detect anomalous
      // gaps (>30s, well above the 2.5s default probe interval) and
      // splice in null markers so connectNulls={false} can break the line.
      return fillRawGaps(pivotRaw(data.rows, targetsToChart));
    }
    // Aggregated tiers know their bucket size — generate a complete
    // grid of buckets across the window and fill missing slots with
    // empty rows that resolve to undefined for every series, which
    // recharts skips when connectNulls is off.
    const interval = BUCKET_INTERVAL_MS[data.granularity];
    return fillBucketGaps(
      pivotAggregated(data.rows, targetsToChart, metric),
      fromMs,
      toMs,
      interval,
    );
  }, [data, targetsToChart, fromMs, toMs, metric]);

  // 7 ticks across whatever window the user picked — `generateTimeTicks`
  // rounds the spacing to whole minutes / hours / days so labels stay
  // legible.
  const xTicks = useMemo(() => generateTimeTicks(fromMs, toMs, 7), [fromMs, toMs]);

  return (
    <YStack height={320}>
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={points} margin={{ top: 4, right: 8, bottom: 0, left: -16 }}>
          <CartesianGrid stroke="var(--borderColor)" strokeDasharray="3 3" />
          <XAxis
            dataKey="ts"
            type="number"
            scale="time"
            domain={[fromMs, toMs]}
            ticks={xTicks}
            tickFormatter={fmtTimeShort}
            tick={{ fontSize: 10, fill: 'var(--color9)' }}
            stroke="var(--borderColor)"
          />
          <YAxis
            tick={{ fontSize: 10, fill: 'var(--color9)' }}
            stroke="var(--borderColor)"
            label={{ value: 'ms', angle: -90, position: 'insideLeft', fill: 'var(--color9)', fontSize: 10, offset: 16 }}
            width={42}
          />
          <Tooltip
            content={<ChartTooltip formatLabel={fmtTimeLong} unit="ms" />}
            cursor={{ stroke: 'var(--color8)', strokeWidth: 1, strokeDasharray: '3 3' }}
          />
          {targetsToChart.map((label) => (
            <Line
              key={label}
              type="monotone"
              dataKey={label}
              stroke={getColor(label)}
              strokeWidth={1.5}
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

function ChartEmpty({ headline, detail }: { headline: string; detail: string }) {
  return (
    <YStack height={320} alignItems="center" justifyContent="center" gap="$1">
      <Text fontSize={13} color="$color10" fontWeight="600">
        {headline}
      </Text>
      <Text fontSize={11} color="$color8" textAlign="center" maxWidth={400}>
        {detail}
      </Text>
    </YStack>
  );
}

// ============================================================================
// Summary stats card
// ============================================================================

function SummaryStats({
  data,
  targetsToChart,
}: {
  data: NonNullable<ReturnType<typeof useSamplesQuery>['data']>;
  targetsToChart: string[];
}) {
  const stats = useMemo(() => {
    const out = new Map<string, { count: number; success: number; rtts: number[]; maxMs: number | null }>();
    const empty = () => ({ count: 0, success: 0, rtts: [] as number[], maxMs: null as number | null });
    if (data.granularity === 'raw') {
      for (const r of data.rows as RawSample[]) {
        if (!targetsToChart.includes(r.target_label)) continue;
        const e = out.get(r.target_label) ?? empty();
        e.count++;
        if (r.success) {
          e.success++;
          if (r.rtt_ms != null) {
            e.rtts.push(r.rtt_ms);
            if (e.maxMs == null || r.rtt_ms > e.maxMs) e.maxMs = r.rtt_ms;
          }
        }
        out.set(r.target_label, e);
      }
    } else {
      // Aggregated: counts and success_count come pre-rolled. We can't
      // reconstruct exact percentile distributions but we can compute a
      // sample-count-weighted mean of bucket p50s, which is close enough
      // for a summary card. Max is exact — max-of-bucket-maxes is the
      // true max across the whole window.
      for (const r of data.rows as AggregatedRow[]) {
        if (!targetsToChart.includes(r.target_label)) continue;
        const e = out.get(r.target_label) ?? empty();
        e.count += r.count;
        e.success += r.success_count;
        if (r.rtt_p50_ms != null && r.success_count > 0) {
          // Approximate: treat each success as one observation at the p50.
          for (let i = 0; i < r.success_count; i++) e.rtts.push(r.rtt_p50_ms);
        }
        if (r.rtt_max_ms != null && (e.maxMs == null || r.rtt_max_ms > e.maxMs)) {
          e.maxMs = r.rtt_max_ms;
        }
        out.set(r.target_label, e);
      }
    }
    return out;
  }, [data, targetsToChart]);

  if (stats.size === 0) return null;

  return (
    <Card title="Per-target stats">
      <YStack gap="$2">
        <XStack paddingVertical="$1" paddingHorizontal="$2" gap="$3" alignItems="center">
          <YStack flexBasis={0} flexGrow={1} minWidth={200}>
            <StatHeaderCell>TARGET</StatHeaderCell>
          </YStack>
          <YStack width={90}>
            <InfoLabel
              label="UPTIME"
              explain="Percentage of probes that succeeded in this window. 99%+ is healthy; below 95% is a sign of real problems."
            />
          </YStack>
          <YStack width={80}>
            <InfoLabel
              label="P50"
              explain="Median round-trip time. Half of probes were faster than this, half slower. Typical 'how fast does this feel' number."
            />
          </YStack>
          <YStack width={80}>
            <InfoLabel
              label="P95"
              explain="95% of probes were faster than this. The slow tail — what video calls actually feel during congestion."
            />
          </YStack>
          <YStack width={80}>
            <InfoLabel
              label="P99"
              explain="99% of probes were faster. The worst 1% — usually transient spikes. High P99 means inconsistent network."
            />
          </YStack>
          <YStack width={80}>
            <InfoLabel
              label="MAX"
              explain="Single worst probe in this window. Surfaces rare incidents that even P99 misses on large samples."
            />
          </YStack>
          <YStack width={80}>
            <InfoLabel
              label="COUNT"
              explain="Total probes counted in this window."
            />
          </YStack>
        </XStack>
        {Array.from(stats.entries())
          .sort((a, b) => a[0].localeCompare(b[0]))
          .map(([label, e]) => {
            const sorted = e.rtts.slice().sort((a, b) => a - b);
            const p50 = pct(sorted, 0.5);
            const p95 = pct(sorted, 0.95);
            const p99 = pct(sorted, 0.99);
            const uptime = e.count > 0 ? (e.success / e.count) * 100 : 0;
            // Uptime tiers — semantic colors, not the watchfire amber. Green
            // for "really hitting it" (three-nines and up), $orange10 for
            // "noticeable trouble" (clearly distinct from $color10 muted
            // text — $yellow10 was washing out on the slate background),
            // $red10 for "fire's out." 99% sits in plain $color12 so the
            // table doesn't shout when nothing's actually wrong.
            const uptimeColor =
              uptime >= 99.9
                ? '$green10'
                : uptime >= 99
                  ? '$color12'
                  : uptime >= 95
                    ? '$orange10'
                    : '$red10';
            return (
              <XStack
                key={label}
                paddingVertical="$1.5"
                paddingHorizontal="$2"
                borderRadius="$2"
                gap="$3"
                hoverStyle={{ backgroundColor: '$color3' }}
                alignItems="center"
              >
                <YStack flexBasis={0} flexGrow={1} minWidth={200}>
                  <Text fontSize={12} color="$color12" numberOfLines={1}>
                    {label}
                  </Text>
                </YStack>
                <YStack width={90}>
                  <Text fontSize={12} color={uptimeColor as any} fontWeight="600" className="vigil-num">
                    {uptime.toFixed(2)}%
                  </Text>
                </YStack>
                <YStack width={80}>
                  <Text fontSize={12} color="$color11" className="vigil-num">
                    {p50 == null ? '—' : `${p50.toFixed(1)}ms`}
                  </Text>
                </YStack>
                <YStack width={80}>
                  <Text fontSize={12} color="$color11" className="vigil-num">
                    {p95 == null ? '—' : `${p95.toFixed(1)}ms`}
                  </Text>
                </YStack>
                <YStack width={80}>
                  <Text fontSize={12} color="$color11" className="vigil-num">
                    {p99 == null ? '—' : `${p99.toFixed(1)}ms`}
                  </Text>
                </YStack>
                <YStack width={80}>
                  <Text fontSize={12} color="$color11" className="vigil-num">
                    {e.maxMs == null ? '—' : `${e.maxMs.toFixed(1)}ms`}
                  </Text>
                </YStack>
                <YStack width={80}>
                  <Text fontSize={12} color="$color9" className="vigil-num">
                    {e.count.toLocaleString()}
                  </Text>
                </YStack>
              </XStack>
            );
          })}
      </YStack>
    </Card>
  );
}

function StatHeaderCell({ children }: { children: React.ReactNode }) {
  return (
    <Text fontSize={10} color="$color8" letterSpacing={0.5} fontWeight="600" numberOfLines={1}>
      {children}
    </Text>
  );
}

function MetricChip({
  label,
  explain,
  active,
  onPress,
}: {
  label: string;
  explain: string;
  active: boolean;
  onPress: () => void;
}) {
  return (
    <XStack
      paddingHorizontal="$2"
      paddingVertical="$1"
      borderRadius="$1.5"
      borderWidth={1}
      borderColor={active ? '$accentBackground' : '$borderColor'}
      backgroundColor={active ? '$accentBackground' : 'transparent'}
      cursor="pointer"
      hoverStyle={{ backgroundColor: active ? '$accentBackground' : '$color3' }}
      pressStyle={{ opacity: 0.85 }}
      animation="quick"
      onPress={onPress}
      title={explain}
    >
      <Text
        fontSize={10}
        fontWeight={active ? '600' : '500'}
        color={active ? '$accentColor' : '$color11'}
        letterSpacing={0.4}
      >
        {label}
      </Text>
    </XStack>
  );
}

// ============================================================================
// Helpers
// ============================================================================

function parseTargets(raw: string | null): string[] {
  if (!raw) return [];
  return raw.split(',').filter(Boolean);
}

/**
 * Reads ?from + ?to (preferred) or ?range (legacy) from the URL and returns
 * a TimeRange. Returns null when neither is present so callers can fall
 * back to defaultRange().
 */
function parseRangeFromUrl(searchParams: URLSearchParams): TimeRange | null {
  const fromRaw = searchParams.get('from');
  const toRaw = searchParams.get('to');
  if (fromRaw && toRaw) {
    const f = Number.parseInt(fromRaw, 10);
    const t = Number.parseInt(toRaw, 10);
    if (Number.isFinite(f) && Number.isFinite(t) && t > f) {
      return { fromMs: f, toMs: t };
    }
  }
  const rangeRaw = searchParams.get('range');
  if (rangeRaw) {
    const n = Number.parseInt(rangeRaw, 10);
    if (Number.isFinite(n) && n > 0) {
      const now = Date.now();
      return { fromMs: now - n, toMs: now };
    }
  }
  return null;
}

// Mirrors the backend's pickGranularity in internal/ipc/handlers_samples.go.
// Keep these boundaries in sync — the label here is what the user sees;
// the backend is what actually decides what data comes back.
function predictedGranularity(rangeMs: number): string {
  if (rangeMs <= 30 * 60 * 1000) return 'raw (per-probe)';
  if (rangeMs <= 6 * 60 * 60 * 1000) return '1-minute buckets';
  if (rangeMs <= 7 * 24 * 60 * 60 * 1000) return '5-minute buckets';
  return '1-hour buckets';
}

function humanizeRange(ms: number): string {
  const h = ms / (60 * 60 * 1000);
  if (h < 24) return `${Math.round(h)}h`;
  return `${Math.round(h / 24)}d`;
}

function pivotRaw(rows: RawSample[], targets: string[]): PivotPoint[] {
  const allowed = new Set(targets);
  const byTs = new Map<number, PivotPoint>();
  for (const r of rows) {
    if (!allowed.has(r.target_label) || !r.success || r.rtt_ms == null) continue;
    const t = CycleBucket(r.ts_unix_ms);
    const existing = byTs.get(t) ?? { ts: t };
    existing[r.target_label] = r.rtt_ms;
    byTs.set(t, existing);
  }
  return Array.from(byTs.values()).sort((a, b) => a.ts - b.ts);
}

function pivotAggregated(rows: AggregatedRow[], targets: string[], metric: MetricKey): PivotPoint[] {
  const allowed = new Set(targets);
  const field = AGG_FIELD[metric];
  const byTs = new Map<number, PivotPoint>();
  for (const r of rows) {
    if (!allowed.has(r.target_label)) continue;
    const v = r[field] as number | undefined;
    if (v == null) continue;
    const existing = byTs.get(r.bucket_start_unix_ms) ?? { ts: r.bucket_start_unix_ms };
    existing[r.target_label] = v;
    byTs.set(r.bucket_start_unix_ms, existing);
  }
  return Array.from(byTs.values()).sort((a, b) => a.ts - b.ts);
}

function pct(sorted: number[], q: number): number | null {
  if (sorted.length === 0) return null;
  const idx = Math.min(Math.floor(sorted.length * q), sorted.length - 1);
  return sorted[idx];
}

function fmtTimeShort(ms: number): string {
  const d = new Date(ms);
  const now = Date.now();
  const ageHours = (now - ms) / (60 * 60 * 1000);
  if (ageHours < 24) {
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }
  return d.toLocaleDateString([], { month: 'short', day: 'numeric' });
}

function fmtTimeLong(ms: number): string {
  return new Date(ms).toLocaleString([], {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  });
}
