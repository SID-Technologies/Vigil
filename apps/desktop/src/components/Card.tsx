import type { ReactNode } from 'react';
import { XStack, YStack, Text } from 'tamagui';

interface CardProps {
  title?: string;
  /** Right-aligned slot for the title row — typically status text or an action. */
  trailing?: ReactNode;
  children: ReactNode;
  /** Optional override for inner padding (default $4). */
  paddingProp?: string;
}

/**
 * Reusable section card. Used by every dashboard panel so spacing and
 * typography stay consistent. Mirrors Pugio's card aesthetic but tuned to
 * Night Watch (subtle border, no heavy shadow).
 */
export function Card({ title, trailing, children, paddingProp = '$4' }: CardProps) {
  return (
    <YStack
      backgroundColor="$color2"
      borderWidth={1}
      borderColor="$borderColor"
      borderRadius="$3"
      padding={paddingProp as any}
      gap="$3"
    >
      {(title || trailing) && (
        <XStack justifyContent="space-between" alignItems="center">
          {title ? (
            <Text fontSize={13} color="$color11" fontWeight="600" letterSpacing={0.3}>
              {title}
            </Text>
          ) : (
            <YStack />
          )}
          {trailing}
        </XStack>
      )}
      {children}
    </YStack>
  );
}
