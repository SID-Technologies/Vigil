import { YStack, XStack } from 'tamagui';

interface SkeletonProps {
  width?: number | string;
  height?: number | string;
  borderRadius?: number | string;
  /** Optional override for the shimmer base color. Defaults to $color4. */
  backgroundColor?: string;
}

/**
 * Skeleton — a single rectangular placeholder block.
 *
 * The shimmer keyframe (vigil-skeleton-pulse) is in global.css and respects
 * prefers-reduced-motion. We use a subtle slate-on-slate range so the
 * placeholders read as "loading" without strobing — closer to a quiet glow
 * than a flashing skeleton.
 *
 * Compose these into ChartSkeleton / TableSkeleton / StatRowSkeleton below
 * to mirror the actual UI shape, so the swap to real content doesn't shift
 * layout.
 */
export function Skeleton({
  width = '100%',
  height = 12,
  borderRadius = 4,
  backgroundColor = '$color4',
}: SkeletonProps) {
  return (
    <YStack
      className="vigil-skeleton"
      width={width as any}
      height={height as any}
      borderRadius={borderRadius as any}
      backgroundColor={backgroundColor as any}
      style={{ animation: 'vigil-skeleton-pulse 1.6s ease-in-out infinite' }}
    />
  );
}

/**
 * ChartSkeleton — placeholder shaped like the RTT chart that will replace
 * it: faux Y-axis labels on the left, faux X-axis labels on the bottom, and
 * a horizontal band where the line lives. Same height (220px) as the real
 * chart card so layout doesn't reflow when data lands.
 */
export function ChartSkeleton({ height = 220 }: { height?: number } = {}) {
  return (
    <YStack height={height} paddingVertical="$2" gap="$2">
      <XStack flex={1} gap="$2">
        {/* Y-axis tick labels */}
        <YStack width={32} justifyContent="space-between" paddingVertical="$1">
          <Skeleton width={20} height={8} />
          <Skeleton width={20} height={8} />
          <Skeleton width={20} height={8} />
          <Skeleton width={20} height={8} />
          <Skeleton width={20} height={8} />
        </YStack>

        {/* Plot area — three stacked bands suggesting where the line sits */}
        <YStack flex={1} gap="$3" justifyContent="center">
          <Skeleton height={2} backgroundColor="$color3" />
          <Skeleton height={2} backgroundColor="$color3" />
          <Skeleton height={20} borderRadius={2} />
          <Skeleton height={2} backgroundColor="$color3" />
          <Skeleton height={2} backgroundColor="$color3" />
        </YStack>
      </XStack>

      {/* X-axis labels */}
      <XStack paddingLeft={40} gap="$3" justifyContent="space-between">
        <Skeleton width={40} height={8} />
        <Skeleton width={40} height={8} />
        <Skeleton width={40} height={8} />
        <Skeleton width={40} height={8} />
        <Skeleton width={40} height={8} />
      </XStack>
    </YStack>
  );
}

/**
 * TableRowSkeleton — one row of a stats table with N columns. Widths
 * vary slightly so the placeholder doesn't read as a hard grid.
 */
export function TableRowSkeleton({ columns = 6 }: { columns?: number } = {}) {
  // Pseudo-random widths, but deterministic across renders.
  const widths = ['60%', '40%', '50%', '45%', '55%', '50%'];
  return (
    <XStack gap="$3" paddingVertical="$2" alignItems="center">
      {Array.from({ length: columns }).map((_, i) => (
        <YStack key={i} flex={1}>
          <Skeleton width={widths[i % widths.length]} height={10} />
        </YStack>
      ))}
    </XStack>
  );
}

/**
 * TileSkeleton — placeholder for a target tile in TargetGrid. Mirrors the
 * shape: header row (label + status dot) + RTT number + sparkline strip.
 */
export function TileSkeleton() {
  return (
    <YStack
      padding="$3"
      gap="$2"
      borderRadius="$2"
      borderWidth={1}
      borderColor="$borderColor"
      backgroundColor="$color2"
      minHeight={104}
    >
      <XStack alignItems="center" gap="$2">
        <Skeleton width={8} height={8} borderRadius={999} />
        <Skeleton width="50%" height={11} />
      </XStack>
      <Skeleton width="40%" height={20} />
      <Skeleton width="100%" height={24} borderRadius={2} />
    </YStack>
  );
}

/**
 * RowSkeleton — generic single-row placeholder for outage / target lists.
 * Uses a leading dot, a primary label, and a trailing meta block.
 */
export function RowSkeleton() {
  return (
    <XStack
      gap="$2"
      alignItems="center"
      paddingVertical="$2.5"
      paddingHorizontal="$2.5"
      borderRadius="$2"
      borderWidth={1}
      borderColor="$borderColor"
      backgroundColor="$color2"
    >
      <Skeleton width={8} height={8} borderRadius={999} />
      <YStack flex={1} gap="$1">
        <Skeleton width="35%" height={11} />
      </YStack>
      <Skeleton width={48} height={10} />
      <Skeleton width={64} height={10} />
    </XStack>
  );
}
