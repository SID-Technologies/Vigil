import { useMemo } from 'react';
import { XStack, YStack, Text, Separator } from 'tamagui';

import { Card } from './Card';
import { useOutages, type Outage } from '../hooks/useOutages';

/**
 * Outages over the last 7 days. Open outages float to the top with a red
 * dot; resolved outages render below in muted text with their duration.
 *
 * "Open" = end_ts_unix_ms is null. Outages auto-close on the first
 * successful cycle that follows the failure run, so a healthy network
 * shows zero open outages.
 */
export function OutagesCard() {
  const fromMs = useMemo(() => Date.now() - 7 * 24 * 60 * 60 * 1000, []);
  const toMs = useMemo(() => Date.now(), []);
  const { data, isLoading } = useOutages({ fromMs, toMs });

  const open = (data ?? []).filter((o) => o.end_ts_unix_ms == null);
  const resolved = (data ?? []).filter((o) => o.end_ts_unix_ms != null);

  return (
    <Card
      title="Outages — last 7 days"
      trailing={
        <Text fontSize={11} color="$color8">
          live · 30s refresh
        </Text>
      }
    >
      {isLoading && !data ? (
        <YStack height={80} alignItems="center" justifyContent="center">
          <Text fontSize={11} color="$color8">
            Loading…
          </Text>
        </YStack>
      ) : open.length === 0 && resolved.length === 0 ? (
        <YStack height={80} alignItems="center" justifyContent="center" gap="$1">
          <Text fontSize={13} color="$accentBackground" fontWeight="600">
            No outages detected
          </Text>
          <Text fontSize={11} color="$color8">
            Clean week — every probe stayed within 3 consecutive failures of healthy.
          </Text>
        </YStack>
      ) : (
        <YStack gap="$2">
          {open.map((o) => (
            <OutageRow key={o.id} outage={o} live />
          ))}
          {open.length > 0 && resolved.length > 0 ? <Separator marginVertical="$1" /> : null}
          {resolved.slice(0, 5).map((o) => (
            <OutageRow key={o.id} outage={o} live={false} />
          ))}
          {resolved.length > 5 ? (
            <Text fontSize={10} color="$color8" paddingLeft="$1">
              +{resolved.length - 5} older outage{resolved.length - 5 === 1 ? '' : 's'} — view in History (phase 5)
            </Text>
          ) : null}
        </YStack>
      )}
    </Card>
  );
}

function OutageRow({ outage, live }: { outage: Outage; live: boolean }) {
  const scopeLabel = outage.scope === 'network' ? 'Network' : outage.scope.replace('target:', '');
  const start = new Date(outage.start_ts_unix_ms);
  const end = outage.end_ts_unix_ms ? new Date(outage.end_ts_unix_ms) : null;
  const durationSec = end
    ? Math.round((end.getTime() - start.getTime()) / 1000)
    : Math.round((Date.now() - start.getTime()) / 1000);

  return (
    <XStack alignItems="center" gap="$2" opacity={live ? 1 : 0.7}>
      <YStack
        width={8}
        height={8}
        borderRadius={999}
        backgroundColor={live ? '$red10' : '$color8'}
      />
      <Text fontSize={12} color="$color12" fontWeight={live ? '600' : '400'} flex={1}>
        {scopeLabel}
      </Text>
      <Text fontSize={11} color="$color9">
        {fmtDuration(durationSec)}
      </Text>
      <Text fontSize={11} color="$color8">
        {live ? `since ${start.toLocaleTimeString()}` : start.toLocaleString([], {
          month: 'short',
          day: 'numeric',
          hour: '2-digit',
          minute: '2-digit',
        })}
      </Text>
    </XStack>
  );
}

function fmtDuration(sec: number): string {
  if (sec < 60) return `${sec}s`;
  if (sec < 3600) return `${Math.round(sec / 60)}m`;
  if (sec < 86400) return `${(sec / 3600).toFixed(1)}h`;
  return `${(sec / 86400).toFixed(1)}d`;
}
