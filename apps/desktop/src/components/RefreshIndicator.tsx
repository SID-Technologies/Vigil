import { useEffect, useState } from 'react';
import { ArrowsClockwise } from '@phosphor-icons/react';
import { XStack, Text } from 'tamagui';

interface RefreshIndicatorProps {
  /** Date of the last successful data fetch. */
  lastUpdated: Date | undefined;
  /** Manual refresh callback — invoked when the user clicks the icon. */
  onRefresh: () => void;
  /** When true, indicator shows a spinning icon instead of "Xs ago". */
  isFetching?: boolean;
}

/**
 * Tiny "Updated 12s ago · ↻" pill for chart and list cards. Updates the
 * "ago" text every second so users never wonder if the page is frozen.
 *
 * Click the indicator to manually trigger a refetch (calls onRefresh).
 */
export function RefreshIndicator({ lastUpdated, onRefresh, isFetching }: RefreshIndicatorProps) {
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    const id = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(id);
  }, []);

  const ago = lastUpdated ? Math.max(0, Math.floor((now - lastUpdated.getTime()) / 1000)) : null;

  return (
    <XStack
      gap="$1.5"
      alignItems="center"
      cursor="pointer"
      onPress={onRefresh}
      paddingHorizontal="$1.5"
      paddingVertical="$1"
      borderRadius="$1"
      hoverStyle={{ backgroundColor: '$color3' }}
      pressStyle={{ opacity: 0.7 }}
    >
      <ArrowsClockwise
        size={11}
        color="var(--color9)"
        weight={isFetching ? 'bold' : 'regular'}
        style={isFetching ? { animation: 'vigil-pulse-ring 1.5s linear infinite' } : undefined}
      />
      <Text fontSize={10} color="$color8">
        {isFetching ? 'updating…' : ago != null ? `updated ${fmtAgo(ago)}` : 'never'}
      </Text>
    </XStack>
  );
}

function fmtAgo(sec: number): string {
  if (sec < 60) return `${sec}s ago`;
  if (sec < 3600) return `${Math.floor(sec / 60)}m ago`;
  return `${Math.floor(sec / 3600)}h ago`;
}
