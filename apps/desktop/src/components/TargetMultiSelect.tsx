import { XStack, YStack, Text } from 'tamagui';

import { useColorPalette } from '../hooks/useColorPalette';
import type { Target } from '../lib/ipc';

interface TargetMultiSelectProps {
  allTargets: Target[];
  selected: string[];
  onToggle: (label: string) => void;
  onSetAll: (labels: string[]) => void;
  onClear: () => void;
  /** Optional message shown next to the quick-filter bar when nothing's selected. */
  emptyMessage?: string;
}

/**
 * Reusable multi-select for target labels. Two-row layout:
 *
 *   Row 1: quick-filter chips (All / ICMP / TCP / UDP / Clear) + a status
 *          line ("None selected — chart shows all targets" or "N selected")
 *   Row 2: every target as a click-to-toggle chip, ALWAYS visible.
 *
 * Each chip:
 *   - Colored dot + border matching palette when selected (matches the
 *     Dashboard tile colors so a target is the same color everywhere)
 *   - Click toggles inclusion in the parent's `selected` array
 *
 * Showing per-target chips unconditionally means the user can drill into
 * a specific target without first clicking "All" or any quick-filter —
 * which was previously the only way to surface them.
 */
export function TargetMultiSelect({
  allTargets,
  selected,
  onToggle,
  onSetAll,
  onClear,
  emptyMessage,
}: TargetMultiSelectProps) {
  const { getColor } = useColorPalette();

  const labelsByKind = (kinds: string[]) =>
    allTargets.filter((t) => kinds.includes(t.kind)).map((t) => t.label);

  const status =
    selected.length === 0
      ? emptyMessage ?? 'None selected'
      : `${selected.length} of ${allTargets.length} selected`;

  return (
    <YStack gap="$2">
      <XStack flexWrap="wrap" gap="$1.5" alignItems="center">
        <QuickChip label="All" onPress={() => onSetAll(allTargets.map((t) => t.label))} />
        <QuickChip label="ICMP" onPress={() => onSetAll(labelsByKind(['icmp']))} />
        <QuickChip label="TCP" onPress={() => onSetAll(labelsByKind(['tcp']))} />
        <QuickChip label="UDP" onPress={() => onSetAll(labelsByKind(['udp_dns', 'udp_stun']))} />
        <QuickChip label="Clear" onPress={onClear} muted />
        <Text fontSize={11} color="$color8" marginLeft="$2">
          {status}
        </Text>
      </XStack>

      <XStack flexWrap="wrap" gap="$1.5" alignItems="center">
        {allTargets.map((t) => {
          const active = selected.includes(t.label);
          const color = getColor(t.label);
          return (
            <XStack
              key={t.label}
              gap="$1.5"
              alignItems="center"
              paddingHorizontal="$2"
              paddingVertical="$1"
              borderRadius="$1.5"
              borderWidth={1}
              borderColor={active ? (color as any) : '$borderColor'}
              backgroundColor={active ? '$color3' : 'transparent'}
              cursor="pointer"
              hoverStyle={{ backgroundColor: '$color3', borderColor: active ? (color as any) : '$color8' }}
              pressStyle={{ opacity: 0.85 }}
              onPress={() => onToggle(t.label)}
              animation="quick"
            >
              <XStack
                width={8}
                height={8}
                borderRadius={999}
                backgroundColor={active ? (color as any) : '$color7'}
              />
              <Text
                fontSize={11}
                color={active ? '$color12' : '$color10'}
                fontWeight={active ? '600' : '400'}
              >
                {t.label}
              </Text>
            </XStack>
          );
        })}
      </XStack>
    </YStack>
  );
}

function QuickChip({
  label,
  onPress,
  muted,
}: {
  label: string;
  onPress: () => void;
  muted?: boolean;
}) {
  return (
    <XStack
      paddingHorizontal="$2"
      paddingVertical="$1"
      borderRadius="$1.5"
      borderWidth={1}
      borderColor="$borderColor"
      cursor="pointer"
      hoverStyle={{ backgroundColor: '$color3', borderColor: '$color8' }}
      pressStyle={{ opacity: 0.85 }}
      onPress={onPress}
      animation="quick"
    >
      <Text fontSize={11} color={muted ? '$color8' : '$color11'} fontWeight="500">
        {label}
      </Text>
    </XStack>
  );
}
