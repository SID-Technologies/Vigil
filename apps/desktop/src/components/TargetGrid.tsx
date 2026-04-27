import { useMemo } from 'react';
import { ArrowSquareOut } from '@phosphor-icons/react';
import { useNavigate } from 'react-router-dom';
import { XStack, YStack, Text } from 'tamagui';

import { Card } from './Card';
import { Sparkline } from './Sparkline';
import { useColorPalette } from '../hooks/useColorPalette';
import { useLiveSamples } from '../hooks/useLiveSamples';
import { useProbeCycle, type ProbeResult } from '../hooks/useProbeCycle';

interface TargetGridProps {
  /** Currently selected target labels (for chart inclusion). */
  selectedLabels: string[];
  /** Toggle a single target. */
  onToggle: (label: string) => void;
}

/**
 * Live grid of every probe's recent activity. Each tile is a click-to-toggle
 * filter for the chart above and shows:
 *
 *   - Target label + kind:port
 *   - 5-min success rate (or "live" failure indicator if currently down)
 *   - Mean RTT over last 5 min
 *   - Inline sparkline of last ~120 RTT points
 *   - Colored ring matching the target's chart-line color when selected
 *   - ↗ "open detail" affordance (navigates to /history?target=…, lands in phase 5)
 *
 * The summary stats come from a rolling buffer fed by probe:cycle events
 * (useLiveSamples). No DB query — the page feels live because it IS live.
 *
 * Hint text at the top tells the user what to do — no one should have to
 * guess that tiles are clickable.
 */
export function TargetGrid({ selectedLabels, onToggle }: TargetGridProps) {
  const { latest } = useProbeCycle();
  const { states } = useLiveSamples();
  const { getColor } = useColorPalette();
  const navigate = useNavigate();

  const sorted = useMemo(() => {
    if (!latest?.results) return [];
    return [...latest.results].sort(
      (a, b) =>
        kindOrder(a) - kindOrder(b) || a.target.label.localeCompare(b.target.label),
    );
  }, [latest]);

  return (
    <Card
      title="Per-target status"
      trailing={
        <Text fontSize={11} color="$color8">
          live · click a tile to add it to the chart
        </Text>
      }
    >
      {sorted.length === 0 ? (
        <YStack height={120} alignItems="center" justifyContent="center">
          <Text fontSize={11} color="$color8">
            Awaiting first cycle…
          </Text>
        </YStack>
      ) : (
        <XStack flexWrap="wrap" gap="$2">
          {sorted.map((r) => {
            const live = states.get(r.target.label);
            const isSelected = selectedLabels.includes(r.target.label);
            const color = getColor(r.target.label);
            return (
              <TargetTile
                key={r.target.label}
                result={r}
                successPct={live?.successPct ?? null}
                avgRTT={live?.avgRTTMs ?? null}
                sparkline={live?.successfulRTTs ?? []}
                isSelected={isSelected}
                color={color}
                onToggle={() => onToggle(r.target.label)}
                onOpenDetail={() => navigate(`/history?target=${encodeURIComponent(r.target.label)}`)}
              />
            );
          })}
        </XStack>
      )}
    </Card>
  );
}

interface TargetTileProps {
  result: ProbeResult;
  successPct: number | null;
  avgRTT: number | null;
  sparkline: number[];
  isSelected: boolean;
  color: string;
  onToggle: () => void;
  onOpenDetail: () => void;
}

function TargetTile({
  result,
  successPct,
  avgRTT,
  sparkline,
  isSelected,
  color,
  onToggle,
  onOpenDetail,
}: TargetTileProps) {
  const { target, success, rtt_ms, error } = result;
  const failing = !success;

  return (
    <YStack
      width={186}
      padding="$2.5"
      borderRadius="$2"
      backgroundColor="$color3"
      borderWidth={isSelected ? 2 : 1}
      borderColor={isSelected ? (color as any) : failing ? '$red8' : '$borderColor'}
      gap="$1.5"
      cursor="pointer"
      hoverStyle={{
        backgroundColor: '$color4',
      }}
      pressStyle={{ scale: 0.98 }}
      animation="quick"
      onPress={onToggle}
    >
      <XStack justifyContent="space-between" alignItems="center" gap="$2">
        <Text fontSize={12} color="$color12" fontWeight="600" numberOfLines={1} flex={1}>
          {target.label}
        </Text>
        <YStack
          width={9}
          height={9}
          borderRadius={999}
          backgroundColor={success ? '$accentBackground' : '$red10'}
        />
      </XStack>

      <XStack justifyContent="space-between" alignItems="center">
        <Text fontSize={9} color="$color8" letterSpacing={0.5}>
          {target.kind.toUpperCase()}
          {target.port ? `:${target.port}` : ''}
        </Text>
        {/*
          Open-detail affordance — phosphor icon. stopPropagation so the
          parent tile's onToggle doesn't fire alongside the navigation.
        */}
        <XStack
          padding="$1"
          borderRadius="$1"
          hoverStyle={{ backgroundColor: '$color5' }}
          onPress={(e: any) => {
            e?.stopPropagation?.();
            onOpenDetail();
          }}
        >
          <ArrowSquareOut size={11} color="var(--color8)" />
        </XStack>
      </XStack>

      <XStack alignItems="center" justifyContent="space-between" gap="$1">
        {failing ? (
          <Text fontSize={11} color="$red10" fontWeight="600" numberOfLines={1}>
            {error ?? 'fail'}
          </Text>
        ) : rtt_ms != null ? (
          <Text fontSize={14} color="$color12" fontWeight="600">
            {rtt_ms.toFixed(1)}
            <Text fontSize={10} color="$color9">
              {' '}
              ms
            </Text>
          </Text>
        ) : (
          <Text fontSize={11} color="$color9">
            —
          </Text>
        )}
        <Sparkline values={sparkline} color={color} width={64} height={20} filled={isSelected} />
      </XStack>

      {/* 5-min summary footer — always rendered so layout is stable. */}
      <XStack justifyContent="space-between">
        <Text fontSize={10} color="$color9">
          {successPct == null ? '— %' : `${successPct.toFixed(0)}%`}
          <Text color="$color8"> 5m</Text>
        </Text>
        <Text fontSize={10} color="$color9">
          {avgRTT == null ? '—' : `${avgRTT.toFixed(0)}ms`}
          <Text color="$color8"> avg</Text>
        </Text>
      </XStack>
    </YStack>
  );
}

function kindOrder(r: ProbeResult): number {
  switch (r.target.kind) {
    case 'icmp':
      return 0;
    case 'tcp':
      return 1;
    case 'udp_dns':
      return 2;
    case 'udp_stun':
      return 3;
    default:
      return 99;
  }
}
