import { YStack, Text } from 'tamagui';

interface PlaceholderProps {
  title: string;
  /** One-line description of what this page will be in phase 5. */
  blurb: string;
}

/**
 * Stub for routes whose real implementation lands in phase 5. Keeps the
 * sidebar nav functional without committing to a half-baked UI now.
 */
export function PlaceholderPage({ title, blurb }: PlaceholderProps) {
  return (
    <YStack
      flex={1}
      alignItems="center"
      justifyContent="center"
      padding="$8"
      gap="$3"
    >
      <Text fontSize={32} fontWeight="700" color="$color12" fontFamily="$heading">
        {title}
      </Text>
      <Text fontSize={13} color="$color9" textAlign="center" maxWidth={520}>
        {blurb}
      </Text>
      <Text fontSize={11} color="$color8" letterSpacing={0.5}>
        Coming in phase 5.
      </Text>
    </YStack>
  );
}
