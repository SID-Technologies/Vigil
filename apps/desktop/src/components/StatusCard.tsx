import { XStack, YStack, Text } from 'tamagui';

import { Card } from './Card';
import { PulsingDot } from './PulsingDot';
import { useHealth } from '../hooks/useHealth';
import { useProbeCycle } from '../hooks/useProbeCycle';
import { useOutages } from '../hooks/useOutages';

/**
 * Top-of-dashboard live status card. Combines:
 *   - sidecar handshake (version + commit)
 *   - last probe cycle ok/total + percentage
 *   - count of currently open outages
 *   - pulsing dot tied to cycle counter — visual heartbeat that proves
 *     the page hasn't frozen between 30s data refetches
 *
 * The status dot at the left is the at-a-glance signal — green / amber /
 * red — derived from the most recent cycle's success rate AND any open
 * outage. Open outage trumps cycle pct since outages mean 3+ consecutive
 * failures, which is more meaningful than a single bad cycle.
 */
export function StatusCard() {
  const health = useHealth();
  const { latest, count } = useProbeCycle();
  const openOutages = useOutages({
    fromMs: Date.now() - 7 * 24 * 60 * 60 * 1000,
    toMs: Date.now(),
    onlyOpen: true,
  });

  const tone = computeTone(latest, openOutages.data ?? []);
  const message = computeMessage(latest, openOutages.data ?? []);

  return (
    <Card>
      <XStack alignItems="center" gap="$3">
        {/*
          Pulsing dot — pulseKey driven by the cycle counter. Each new probe
          cycle remounts the ring element, restarting the keyframe. Net effect:
          a subtle radar-ping every ~2.5 seconds.
        */}
        <PulsingDot color={tone.cssColor} size={14} pulseKey={count} />
        <YStack flex={1} gap="$1">
          <Text fontSize={18} color="$color12" fontWeight="600">
            {message.headline}
          </Text>
          <Text fontSize={12} color="$color9">
            {message.detail}
          </Text>
        </YStack>
        <YStack alignItems="flex-end" gap="$0.5">
          <Text fontSize={11} color="$color9">
            sidecar
          </Text>
          <Text fontSize={11} color="$color11" fontWeight="600" className="vigil-num">
            {health.isError
              ? 'disconnected'
              : health.data
                ? `v${health.data.version}${health.data.commit ? ` (${health.data.commit})` : ''}`
                : '…'}
          </Text>
          <Text fontSize={10} color="$color8" className="vigil-num">
            {count} cycle{count === 1 ? '' : 's'} since open
          </Text>
        </YStack>
      </XStack>
    </Card>
  );
}

interface Tone {
  /** CSS color string for the pulse (must be a hex/rgb/var, NOT a Tamagui token). */
  cssColor: string;
}

function computeTone(
  latest: ReturnType<typeof useProbeCycle>['latest'],
  openOutages: { id: string }[],
): Tone {
  if (openOutages.length > 0) return { cssColor: '#f85149' };
  if (!latest) return { cssColor: '#8b949e' };
  const pct = latest.total > 0 ? latest.ok / latest.total : 0;
  if (pct >= 1) return { cssColor: '#e0a458' }; // watchfire amber
  if (pct >= 0.8) return { cssColor: '#d29922' };
  return { cssColor: '#f85149' };
}

function computeMessage(
  latest: ReturnType<typeof useProbeCycle>['latest'],
  openOutages: { scope: string }[],
): { headline: string; detail: string } {
  if (openOutages.length > 0) {
    const networkOpen = openOutages.some((o) => o.scope === 'network');
    if (networkOpen) {
      return {
        headline: 'Network outage in progress',
        detail: 'Every probe is failing. Check your modem / router / ISP.',
      };
    }
    return {
      headline: `${openOutages.length} target outage${openOutages.length === 1 ? '' : 's'} in progress`,
      detail: openOutages.map((o) => o.scope.replace('target:', '')).join(', '),
    };
  }
  if (!latest) {
    return {
      headline: 'Settling in',
      detail: 'First report in just a moment — Vigil checks every 2.5 seconds.',
    };
  }
  const pct = latest.total > 0 ? Math.round((latest.ok / latest.total) * 100) : 0;
  if (pct >= 100) {
    return {
      headline: 'All systems nominal',
      detail: `Just checked ${latest.total} targets — all reachable · ${new Date(latest.ts_unix_ms).toLocaleTimeString()}`,
    };
  }
  return {
    headline: `${latest.fail} of ${latest.total} targets unreachable just now`,
    detail: `${pct}% reachability. If this lasts 3 cycles in a row, Vigil records it as an outage.`,
  };
}
