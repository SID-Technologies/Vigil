import { useState } from 'react';
import { CaretDown, CaretRight } from '@phosphor-icons/react';
import { XStack, YStack, Text, Separator } from 'tamagui';

import { Card } from '../components/Card';
import { PageHeader } from '../components/PageHeader';
import { TimeRangePicker, defaultRange, type TimeRange } from '../components/TimeRangePicker';
import { useOutages, type Outage } from '../hooks/useOutages';
import { useTargets } from '../hooks/useTargets';

const SEVEN_DAYS_MS = 7 * 24 * 60 * 60 * 1000;

type ScopeFilter = 'all' | 'network' | string; // string = "target:<label>"

/**
 * Outages — full historical timeline with scope filter and expandable
 * details. Live-updates via outage:start / outage:end (handled inside
 * useOutages).
 *
 * Each row collapses by default. Clicking expands to show:
 *   - exact start / end (or "ongoing")
 *   - duration in human time
 *   - error breakdown — which error codes contributed to the failure run
 *
 * Filters at the top:
 *   - Time range (1h–30d)
 *   - Scope: All, Network, or any specific target
 */
export function OutagesPage() {
  const [range, setRange] = useState<TimeRange>(() => defaultRange(SEVEN_DAYS_MS));
  const [scope, setScope] = useState<ScopeFilter>('all');
  const [expanded, setExpanded] = useState<Set<string>>(new Set());

  const { fromMs, toMs } = range;

  const targets = useTargets();
  const outages = useOutages({
    fromMs,
    toMs,
    scope: scope === 'all' ? undefined : scope,
  });

  const all = outages.data ?? [];
  const open = all.filter((o) => o.end_ts_unix_ms == null);
  const resolved = all.filter((o) => o.end_ts_unix_ms != null);

  const toggleExpand = (id: string) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  return (
    <YStack flex={1}>
      <PageHeader
        title="Outages"
        blurb="Every detected reachability gap (3+ consecutive failures of one target or every probe) recorded since the sidecar started. Pruned: never."
      />
      <YStack padding="$4" gap="$3" maxWidth={1100} width="100%" alignSelf="center">
        <Card>
          <YStack gap="$3">
            <XStack gap="$3" alignItems="center" flexWrap="wrap">
              <Text fontSize={11} color="$color10" letterSpacing={0.4} fontWeight="600">
                WINDOW
              </Text>
              <TimeRangePicker value={range} onChange={setRange} />
            </XStack>
            <Separator />
            <XStack gap="$2" alignItems="center" flexWrap="wrap">
              <Text fontSize={11} color="$color10" letterSpacing={0.4} fontWeight="600">
                SCOPE
              </Text>
              <ScopeChip label="All" active={scope === 'all'} onPress={() => setScope('all')} />
              <ScopeChip
                label="Network"
                active={scope === 'network'}
                onPress={() => setScope('network')}
              />
              {(targets.data ?? []).map((t) => {
                const v = `target:${t.label}`;
                return (
                  <ScopeChip
                    key={t.label}
                    label={t.label}
                    active={scope === v}
                    onPress={() => setScope(v)}
                    small
                  />
                );
              })}
            </XStack>
          </YStack>
        </Card>

        {outages.isLoading && !outages.data ? (
          <Card>
            <YStack height={120} alignItems="center" justifyContent="center">
              <Text fontSize={12} color="$color9">Loading…</Text>
            </YStack>
          </Card>
        ) : all.length === 0 ? (
          <Card>
            <YStack height={120} alignItems="center" justifyContent="center" gap="$1">
              <Text fontSize={14} color="$accentBackground" fontWeight="600">
                No outages recorded
              </Text>
              <Text fontSize={11} color="$color8" textAlign="center" maxWidth={420}>
                Either the sidecar hasn't been running long enough OR your network has been
                stable. Both are good problems to have.
              </Text>
            </YStack>
          </Card>
        ) : (
          <>
            {open.length > 0 && (
              <Card title="Ongoing">
                <YStack gap="$1">
                  {open.map((o) => (
                    <OutageRow
                      key={o.id}
                      outage={o}
                      expanded={expanded.has(o.id)}
                      onToggle={() => toggleExpand(o.id)}
                      live
                    />
                  ))}
                </YStack>
              </Card>
            )}
            <Card title={`Resolved — ${resolved.length}`}>
              <YStack gap="$1">
                {resolved.length === 0 ? (
                  <Text fontSize={11} color="$color8" padding="$2">
                    No resolved outages in this window.
                  </Text>
                ) : (
                  resolved.map((o) => (
                    <OutageRow
                      key={o.id}
                      outage={o}
                      expanded={expanded.has(o.id)}
                      onToggle={() => toggleExpand(o.id)}
                    />
                  ))
                )}
              </YStack>
            </Card>
          </>
        )}
      </YStack>
    </YStack>
  );
}

function ScopeChip({
  label,
  active,
  onPress,
  small,
}: {
  label: string;
  active: boolean;
  onPress: () => void;
  small?: boolean;
}) {
  return (
    <XStack
      paddingHorizontal={small ? '$1.5' : '$2'}
      paddingVertical="$1"
      borderRadius="$1.5"
      borderWidth={1}
      borderColor={active ? '$accentBackground' : '$borderColor'}
      backgroundColor={active ? '$accentBackground' : 'transparent'}
      cursor="pointer"
      hoverStyle={{ backgroundColor: active ? '$accentBackground' : '$color3' }}
      pressStyle={{ opacity: 0.85 }}
      onPress={onPress}
      animation="quick"
    >
      <Text fontSize={small ? 10 : 11} color={active ? '$accentColor' : '$color11'} fontWeight={active ? '600' : '500'}>
        {label}
      </Text>
    </XStack>
  );
}

function OutageRow({
  outage,
  expanded,
  onToggle,
  live,
}: {
  outage: Outage;
  expanded: boolean;
  onToggle: () => void;
  live?: boolean;
}) {
  const scopeLabel = outage.scope === 'network' ? 'Network' : outage.scope.replace('target:', '');
  const start = new Date(outage.start_ts_unix_ms);
  const end = outage.end_ts_unix_ms ? new Date(outage.end_ts_unix_ms) : null;
  const durationSec = end
    ? Math.round((end.getTime() - start.getTime()) / 1000)
    : Math.round((Date.now() - start.getTime()) / 1000);

  return (
    <YStack
      borderRadius="$2"
      borderWidth={1}
      borderColor={live ? '$red8' : '$borderColor'}
      backgroundColor="$color3"
      padding="$2.5"
      gap="$2"
      cursor="pointer"
      hoverStyle={{ backgroundColor: '$color4' }}
      animation="quick"
      onPress={onToggle}
    >
      <XStack alignItems="center" gap="$2">
        {expanded ? (
          <CaretDown size={12} color="var(--color9)" />
        ) : (
          <CaretRight size={12} color="var(--color9)" />
        )}
        <YStack
          width={8}
          height={8}
          borderRadius={999}
          backgroundColor={live ? '$red10' : '$color8'}
        />
        <Text fontSize={13} color="$color12" fontWeight={live ? '600' : '500'} flex={1}>
          {scopeLabel}
        </Text>
        <Text fontSize={11} color="$color9">
          {outage.consecutive_failures} consecutive failures
        </Text>
        <Text fontSize={11} color="$color11" fontWeight="600">
          {fmtDuration(durationSec)}
        </Text>
        <Text fontSize={11} color="$color8">
          {live ? 'since' : ''}{' '}
          {start.toLocaleString([], {
            month: 'short',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit',
          })}
        </Text>
      </XStack>
      {expanded ? (
        <YStack
          paddingLeft="$5"
          paddingTop="$1"
          gap="$2"
          borderTopWidth={1}
          borderTopColor="$borderColor"
          marginTop="$1"
        >
          <XStack gap="$3" flexWrap="wrap">
            <DetailField label="Start (UTC)" value={start.toISOString()} />
            <DetailField
              label={live ? 'Status' : 'End (UTC)'}
              value={end ? end.toISOString() : 'ongoing'}
              tone={live ? 'warn' : undefined}
            />
            <DetailField label="Duration" value={fmtDurationLong(durationSec)} />
          </XStack>
          {outage.errors && Object.keys(outage.errors).length > 0 ? (
            <YStack gap="$1.5">
              <Text fontSize={10} color="$color8" letterSpacing={0.5} fontWeight="600">
                ERROR BREAKDOWN
              </Text>
              <XStack gap="$1.5" flexWrap="wrap">
                {Object.entries(outage.errors)
                  .sort((a, b) => b[1] - a[1])
                  .map(([code, count]) => (
                    <XStack
                      key={code}
                      paddingHorizontal="$2"
                      paddingVertical="$1"
                      borderRadius="$1"
                      backgroundColor="$color2"
                      borderWidth={1}
                      borderColor="$borderColor"
                      gap="$1.5"
                    >
                      <Text fontSize={11} color="$color11" fontFamily="$body">
                        {code}
                      </Text>
                      <Text fontSize={11} color="$color8">
                        ×{count}
                      </Text>
                    </XStack>
                  ))}
              </XStack>
            </YStack>
          ) : null}
        </YStack>
      ) : null}
    </YStack>
  );
}

function DetailField({
  label,
  value,
  tone,
}: {
  label: string;
  value: string;
  tone?: 'warn';
}) {
  return (
    <YStack gap="$0.5">
      <Text fontSize={9} color="$color8" letterSpacing={0.5} fontWeight="600">
        {label.toUpperCase()}
      </Text>
      <Text
        fontSize={11}
        color={tone === 'warn' ? '$red10' : '$color11'}
        fontFamily="$body"
      >
        {value}
      </Text>
    </YStack>
  );
}

function fmtDuration(sec: number): string {
  if (sec < 60) return `${sec}s`;
  if (sec < 3600) return `${Math.round(sec / 60)}m`;
  if (sec < 86400) return `${(sec / 3600).toFixed(1)}h`;
  return `${(sec / 86400).toFixed(1)}d`;
}

function fmtDurationLong(sec: number): string {
  const h = Math.floor(sec / 3600);
  const m = Math.floor((sec % 3600) / 60);
  const s = sec % 60;
  if (h > 0) return `${h}h ${m}m ${s}s`;
  if (m > 0) return `${m}m ${s}s`;
  return `${s}s`;
}
