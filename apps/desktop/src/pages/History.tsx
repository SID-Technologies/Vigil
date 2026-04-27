import { useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { Export } from '@phosphor-icons/react';
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
import { GenerateReportModal } from '../components/GenerateReportModal';
import { PageHeader } from '../components/PageHeader';
import { TargetMultiSelect } from '../components/TargetMultiSelect';
import { TimeRangePicker, defaultRange, type TimeRange } from '../components/TimeRangePicker';
import { useColorPalette } from '../hooks/useColorPalette';
import { useSamplesQuery } from '../hooks/useSamplesQuery';
import { useTargets } from '../hooks/useTargets';
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
export function HistoryPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const targets = useTargets();
  const { getColor } = useColorPalette();

  // Initialize from URL once. Keep selection + range in component state,
  // then sync to URL on change so back/forward navigation works.
  const [selected, setSelected] = useState<string[]>(() => parseTargets(searchParams.get('target')));
  const [range, setRange] = useState<TimeRange>(() => parseRangeFromUrl(searchParams) ?? defaultRange());
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

  const targetsToChart = selected.length > 0 ? selected : (targets.data ?? []).map((t) => t.label);

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
            icon={<Export size={14} color="var(--accentColor)" />}
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

      <YStack padding="$4" gap="$3" maxWidth={1300} width="100%" alignSelf="center">
        <Card>
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
                allTargets={targets.data ?? []}
                selected={selected}
                onToggle={(label) =>
                  setSelected((prev) =>
                    prev.includes(label) ? prev.filter((l) => l !== label) : [...prev, label],
                  )
                }
                onSetAll={setSelected}
                onClear={() => setSelected([])}
                emptyMessage="None selected — chart will show all targets."
              />
            </YStack>
          </YStack>
        </Card>

        <Card title={`RTT — ${humanizeRange(rangeMs)}`}>
          {samples.isLoading && !samples.data ? (
            <ChartEmpty headline="Loading…" detail={`Querying samples from the last ${humanizeRange(rangeMs)}.`} />
          ) : samples.isError ? (
            <ChartEmpty headline="Error" detail="Sample query failed." />
          ) : !samples.data || samples.data.rows.length === 0 ? (
            <ChartEmpty
              headline="No data in this window"
              detail="Either the sidecar hasn't been running long enough or every probe was disabled. Try a longer window."
            />
          ) : (
            <HistoryChart
              data={samples.data}
              targetsToChart={targetsToChart}
              getColor={getColor}
            />
          )}
        </Card>

        {samples.data ? (
          <SummaryStats data={samples.data} targetsToChart={targetsToChart} />
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
}: {
  data: NonNullable<ReturnType<typeof useSamplesQuery>['data']>;
  targetsToChart: string[];
  getColor: (label: string) => string;
}) {
  const points = useMemo<PivotPoint[]>(() => {
    if (data.granularity === 'raw') {
      return pivotRaw(data.rows, targetsToChart);
    }
    return pivotAggregated(data.rows, targetsToChart);
  }, [data, targetsToChart]);

  return (
    <YStack height={320}>
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={points} margin={{ top: 4, right: 8, bottom: 0, left: -16 }}>
          <CartesianGrid stroke="var(--borderColor)" strokeDasharray="3 3" />
          <XAxis
            dataKey="ts"
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
            contentStyle={{
              background: 'var(--color2)',
              border: '1px solid var(--borderColor)',
              borderRadius: 6,
              fontSize: 11,
            }}
            labelFormatter={fmtTimeLong}
            formatter={(v: number, name: string) => [`${v.toFixed(2)} ms`, name]}
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
              connectNulls
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
    const out = new Map<string, { count: number; success: number; rtts: number[] }>();
    if (data.granularity === 'raw') {
      for (const r of data.rows as RawSample[]) {
        if (!targetsToChart.includes(r.target_label)) continue;
        const e = out.get(r.target_label) ?? { count: 0, success: 0, rtts: [] };
        e.count++;
        if (r.success) {
          e.success++;
          if (r.rtt_ms != null) e.rtts.push(r.rtt_ms);
        }
        out.set(r.target_label, e);
      }
    } else {
      // Aggregated: counts and success_count come pre-rolled. We can't
      // reconstruct exact percentile distributions but we can compute a
      // sample-count-weighted mean of bucket p50s, which is close enough
      // for a summary card.
      for (const r of data.rows as AggregatedRow[]) {
        if (!targetsToChart.includes(r.target_label)) continue;
        const e = out.get(r.target_label) ?? { count: 0, success: 0, rtts: [] };
        e.count += r.count;
        e.success += r.success_count;
        if (r.rtt_p50_ms != null && r.success_count > 0) {
          // Approximate: treat each success as one observation at the p50.
          for (let i = 0; i < r.success_count; i++) e.rtts.push(r.rtt_p50_ms);
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
        <XStack paddingVertical="$1" paddingHorizontal="$2" gap="$3">
          <Text flex={2} fontSize={10} color="$color8" letterSpacing={0.5} fontWeight="600">
            TARGET
          </Text>
          <Text flex={1} fontSize={10} color="$color8" letterSpacing={0.5} fontWeight="600">
            UPTIME
          </Text>
          <Text flex={1} fontSize={10} color="$color8" letterSpacing={0.5} fontWeight="600">
            P50
          </Text>
          <Text flex={1} fontSize={10} color="$color8" letterSpacing={0.5} fontWeight="600">
            P95
          </Text>
          <Text flex={1} fontSize={10} color="$color8" letterSpacing={0.5} fontWeight="600">
            P99
          </Text>
          <Text flex={1} fontSize={10} color="$color8" letterSpacing={0.5} fontWeight="600">
            COUNT
          </Text>
        </XStack>
        {Array.from(stats.entries())
          .sort((a, b) => a[0].localeCompare(b[0]))
          .map(([label, e]) => {
            const sorted = e.rtts.slice().sort((a, b) => a - b);
            const p50 = pct(sorted, 0.5);
            const p95 = pct(sorted, 0.95);
            const p99 = pct(sorted, 0.99);
            const uptime = e.count > 0 ? (e.success / e.count) * 100 : 0;
            return (
              <XStack
                key={label}
                paddingVertical="$1.5"
                paddingHorizontal="$2"
                borderRadius="$2"
                gap="$3"
                hoverStyle={{ backgroundColor: '$color3' }}
              >
                <Text flex={2} fontSize={12} color="$color12" numberOfLines={1}>
                  {label}
                </Text>
                <Text flex={1} fontSize={12} color={uptime >= 99.5 ? '$accentBackground' : uptime >= 95 ? '$yellow10' : '$red10'} fontWeight="600">
                  {uptime.toFixed(2)}%
                </Text>
                <Text flex={1} fontSize={12} color="$color11">
                  {p50 == null ? '—' : `${p50.toFixed(1)}ms`}
                </Text>
                <Text flex={1} fontSize={12} color="$color11">
                  {p95 == null ? '—' : `${p95.toFixed(1)}ms`}
                </Text>
                <Text flex={1} fontSize={12} color="$color11">
                  {p99 == null ? '—' : `${p99.toFixed(1)}ms`}
                </Text>
                <Text flex={1} fontSize={12} color="$color9">
                  {e.count.toLocaleString()}
                </Text>
              </XStack>
            );
          })}
      </YStack>
    </Card>
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

function predictedGranularity(rangeMs: number): string {
  if (rangeMs <= 2 * 60 * 60 * 1000) return 'raw (per-probe)';
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
    const existing = byTs.get(r.ts_unix_ms) ?? { ts: r.ts_unix_ms };
    existing[r.target_label] = r.rtt_ms;
    byTs.set(r.ts_unix_ms, existing);
  }
  return Array.from(byTs.values()).sort((a, b) => a.ts - b.ts);
}

function pivotAggregated(rows: AggregatedRow[], targets: string[]): PivotPoint[] {
  const allowed = new Set(targets);
  const byTs = new Map<number, PivotPoint>();
  for (const r of rows) {
    if (!allowed.has(r.target_label) || r.rtt_p50_ms == null) continue;
    const existing = byTs.get(r.bucket_start_unix_ms) ?? { ts: r.bucket_start_unix_ms };
    existing[r.target_label] = r.rtt_p50_ms;
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
