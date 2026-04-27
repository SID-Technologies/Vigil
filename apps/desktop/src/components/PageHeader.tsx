import type { ReactNode } from 'react';
import { XStack, YStack, Text } from 'tamagui';

interface PageHeaderProps {
  title: string;
  blurb?: string;
  /** Right-aligned slot — typically action buttons. */
  trailing?: ReactNode;
}

/**
 * Top-of-page header used by every non-Dashboard page. Consistent typography
 * and spacing keeps the app feeling unified — every page reads "Vigil",
 * not "five different React apps in a trenchcoat."
 */
export function PageHeader({ title, blurb, trailing }: PageHeaderProps) {
  return (
    <XStack
      paddingHorizontal="$4"
      paddingTop="$4"
      paddingBottom="$3"
      borderBottomWidth={1}
      borderBottomColor="$borderColor"
      justifyContent="space-between"
      alignItems="flex-end"
      gap="$3"
    >
      <YStack gap="$1" flex={1}>
        <Text fontSize={24} fontWeight="700" color="$color12" fontFamily="$heading">
          {title}
        </Text>
        {blurb ? (
          <Text fontSize={12} color="$color9">
            {blurb}
          </Text>
        ) : null}
      </YStack>
      {trailing}
    </XStack>
  );
}
