import type { ReactNode } from 'react';
import { YStack, Text } from 'tamagui';

interface SectionProps {
  title: string;
  /** Optional sub-headline shown below the title in muted text. */
  description?: string;
  children: ReactNode;
}

/**
 * Borderless titled section with a thin underline divider — for pages
 * where stacking 5+ identically-bordered cards feels monotonous (Settings
 * is the worst offender).
 *
 * Visual: small uppercase title, optional description, divider, content.
 * Tighter than a Card; reads like a GitHub settings page section. Use
 * `Section` for groupings on the *same* page; use `Card` when content is
 * independent enough to warrant a visual container.
 */
export function Section({ title, description, children }: SectionProps) {
  return (
    <YStack paddingTop="$4" gap="$3">
      <YStack
        gap="$1"
        paddingBottom="$2"
        borderBottomWidth={1}
        borderBottomColor="$borderColor"
      >
        <Text
          fontSize={11}
          color="$color10"
          letterSpacing={0.6}
          fontWeight="600"
          textTransform="uppercase"
        >
          {title}
        </Text>
        {description ? (
          <Text fontSize={11} color="$color8">
            {description}
          </Text>
        ) : null}
      </YStack>
      <YStack gap="$3">{children}</YStack>
    </YStack>
  );
}
