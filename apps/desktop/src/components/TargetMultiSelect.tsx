import { XStack, Text } from 'tamagui';

import { useColorPalette } from '../hooks/useColorPalette';
import type { Target } from '../lib/ipc';

interface TargetMultiSelectProps {
  allTargets: Target[];
  selected: string[];
  onToggle: (label: string) => void;
  onSetAll: (labels: string[]) => void;
  onClear: () => void;
  /** Optional emptyMessage shown when nothing's selected. */
  emptyMessage?: string;
}

/**
 * Reusable multi-select for target labels. Used by History page (shared
 * with the dashboard, but here as a chip cloud you can clear/replace
 * wholesale).
 *
 * Each chip:
 *   - colored ring matches palette color (consistent with Dashboard tiles)
 *   - click toggles inclusion
 *
 * Quick-select bar at the top: All / by-kind / Clear.
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

  return (
    <XStack flexWrap="wrap" gap="$1.5" alignItems="center">
      <QuickChip label="All" onPress={() => onSetAll(allTargets.map((t) => t.label))} />
      <QuickChip label="ICMP" onPress={() => onSetAll(labelsByKind(['icmp']))} />
      <QuickChip label="TCP" onPress={() => onSetAll(labelsByKind(['tcp']))} />
      <QuickChip label="UDP" onPress={() => onSetAll(labelsByKind(['udp_dns', 'udp_stun']))} />
      <QuickChip label="Clear" onPress={onClear} muted />

      <Text fontSize={11} color="$color8" marginHorizontal="$2">
        ·
      </Text>

      {selected.length === 0 && emptyMessage ? (
        <Text fontSize={11} color="$color8">
          {emptyMessage}
        </Text>
      ) : (
        allTargets.map((t) => {
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
              borderColor={active ? color : '$borderColor'}
              backgroundColor={active ? '$color3' : 'transparent'}
              cursor="pointer"
              hoverStyle={{ backgroundColor: '$color3' }}
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
        })
      )}
    </XStack>
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
