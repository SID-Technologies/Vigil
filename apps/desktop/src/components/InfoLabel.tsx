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
 * The tooltip uses the native browser `title` attribute. Less
 * customizable than a popover but: zero JS, no positioning bugs, works
 * offline, screen-reader-friendly. The cost is a ~500ms hover delay
 * before it shows — acceptable for a "what does P95 mean?" lookup.
 */
export function InfoLabel({ label, explain }: InfoLabelProps) {
  return (
    <XStack
      gap="$1"
      alignItems="center"
      cursor="help"
      // @ts-expect-error — Tamagui forwards `title` to the underlying DOM element
      title={explain}
    >
      <Text fontSize={10} color="$color8" letterSpacing={0.5} fontWeight="600" numberOfLines={1}>
        {label}
      </Text>
      <Info size={10} color="var(--color8)" />
    </XStack>
  );
}
