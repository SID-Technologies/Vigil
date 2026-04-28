import { Info } from '@phosphor-icons/react';
import { XStack, Text } from 'tamagui';

interface InfoLabelProps {
  /** The visible label (uppercase, small caps style by default). */
  label: string;
  /** Plain-language explanation shown on hover. */
  explain: string;
}

/**
 * A column header with a small ⓘ icon and a hover tooltip explaining what
 * the term means in plain language. Used on stat tables (P50 / P95 / P99
 * etc) where the abbreviation is industry-standard but opaque to users
 * who aren't network engineers.
 *
 * The tooltip uses the native browser `title` attribute, but on a wrapping
 * <span> — Tamagui doesn't reliably forward unknown DOM attributes through
 * its styled components on web, so putting `title` on an XStack silently
 * dropped it. The native span is invisible (display: contents) so it
 * doesn't disturb layout.
 *
 * Trade-off vs a custom popover: ~500ms hover delay before it appears, no
 * styling control. Worth it: zero JS, no positioning bugs, screen-reader
 * support comes for free.
 */
export function InfoLabel({ label, explain }: InfoLabelProps) {
  return (
    <span title={explain} style={{ display: 'inline-flex', cursor: 'help' }}>
      <XStack gap="$1" alignItems="center">
        <Text fontSize={10} color="$color8" letterSpacing={0.5} fontWeight="600" numberOfLines={1}>
          {label}
        </Text>
        <Info size={10} color="var(--color8)" />
      </XStack>
    </span>
  );
}
