import { YStack, XStack } from 'tamagui';

import { StatusCard } from '../components/StatusCard';
import { RTTChart } from '../components/RTTChart';
import { TargetGrid } from '../components/TargetGrid';
import { OutagesCard } from '../components/OutagesCard';
import { useTargetSelection } from '../hooks/useTargetSelection';

/**
 * Dashboard composition. Stack from highest-information-density top-down:
 *
 *   1. StatusCard — "is it ok right now"
 *   2. RTTChart   — "is it getting slower" (selection-driven)
 *   3. Two-column row: TargetGrid + OutagesCard
 *
 * Selection state is owned here and prop-drilled to RTTChart and TargetGrid
 * so they stay in sync. Persistence lives inside useTargetSelection
 * (localStorage), so reload restores the user's last view.
 */
export function DashboardPage() {
  const selection = useTargetSelection();

  return (
    <YStack gap="$3" padding="$4" maxWidth={1200} width="100%" alignSelf="center">
      <StatusCard />
      <RTTChart
        selectedLabels={selection.labels}
        onSetAll={selection.setAll}
        onClear={selection.clear}
      />
      <XStack gap="$3" flexWrap="wrap">
        <YStack flex={1} minWidth={420}>
          <TargetGrid selectedLabels={selection.labels} onToggle={selection.toggle} />
        </YStack>
        <YStack flex={1} minWidth={360}>
          <OutagesCard />
        </YStack>
      </XStack>
    </YStack>
  );
}
