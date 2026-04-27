import { XStack, Text } from 'tamagui';

import type { Target } from '../lib/ipc';

interface ChartFilterBarProps {
  /** All available targets (used for the "All" / per-kind quick selects). */
  allTargets: Target[];
  /** Currently selected target labels. */
  selected: string[];
  /** Replace selection wholesale. */
  onSetAll: (labels: string[]) => void;
  /** Clear selection (back to median view). */
  onClear: () => void;
}

/**
 * Quick-select buttons above the RTT chart. "All" selects every target;
 * the kind buttons (ICMP / TCP / UDP) select only that kind's targets;
 * Clear empties selection back to the median view.
 *
 * UDP combines udp_dns + udp_stun — separating them is more granular than
 * dashboard-level use cases warrant. (Phase 5 history page can offer
 * finer-grained kind filters.)
 */
export function ChartFilterBar({ allTargets, selected, onSetAll, onClear }: ChartFilterBarProps) {
  const allLabels = allTargets.map((t) => t.label);
  const labelsByKind = (kinds: string[]) =>
    allTargets.filter((t) => kinds.includes(t.kind)).map((t) => t.label);

  return (
    <XStack gap="$1.5" alignItems="center" flexWrap="wrap">
      <FilterChip
        label="All"
        active={selected.length === allLabels.length && allLabels.length > 0}
        onPress={() => onSetAll(allLabels)}
      />
      <FilterChip
        label="ICMP"
        active={onlyContains(selected, labelsByKind(['icmp']))}
        onPress={() => onSetAll(labelsByKind(['icmp']))}
      />
      <FilterChip
        label="TCP"
        active={onlyContains(selected, labelsByKind(['tcp']))}
        onPress={() => onSetAll(labelsByKind(['tcp']))}
      />
      <FilterChip
        label="UDP"
        active={onlyContains(selected, labelsByKind(['udp_dns', 'udp_stun']))}
        onPress={() => onSetAll(labelsByKind(['udp_dns', 'udp_stun']))}
      />
      <FilterChip label="Clear" active={false} onPress={onClear} muted />
      <Text fontSize={11} color="$color8" marginLeft="$2">
        {selected.length === 0
          ? 'showing the median across all targets'
          : `${selected.length} target${selected.length === 1 ? '' : 's'} selected`}
      </Text>
    </XStack>
  );
}

function FilterChip({
  label,
  active,
  onPress,
  muted,
}: {
  label: string;
  active: boolean;
  onPress: () => void;
  muted?: boolean;
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
      hoverStyle={{
        borderColor: active ? '$accentBackground' : '$color8',
        backgroundColor: active ? '$accentBackground' : '$color3',
      }}
      pressStyle={{ opacity: 0.8 }}
      animation="quick"
      onPress={onPress}
    >
      <Text
        fontSize={11}
        fontWeight={active ? '600' : '500'}
        color={active ? '$accentColor' : muted ? '$color8' : '$color11'}
      >
        {label}
      </Text>
    </XStack>
  );
}

/**
 * True iff `selected` is exactly the set `target` (same length, same membership).
 * Used to highlight the "all ICMP / TCP / UDP" buttons when their kind set is
 * exactly what's currently selected.
 */
function onlyContains(selected: string[], target: string[]): boolean {
  if (selected.length !== target.length || target.length === 0) return false;
  const set = new Set(target);
  for (const s of selected) {
    if (!set.has(s)) return false;
  }
  return true;
}
