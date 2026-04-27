import type { ReactNode } from 'react';
import { XStack, YStack, Text } from 'tamagui';

export type CardVariant = 'solid' | 'quiet';

interface CardProps {
  title?: string;
  /** Right-aligned slot for the title row — typically status text or an action. */
  trailing?: ReactNode;
  children: ReactNode;
  /** Optional override for inner padding (default $4 for solid, $0 for quiet). */
  paddingProp?: string;
  /**
   * Visual treatment.
   *
   * - `solid` (default): border + filled background. For primary content
   *    that the user comes to the page to read (charts, tables, lists).
   * - `quiet`: no border, transparent background, no padding. For
   *    secondary controls (filter bars, status banners) that shouldn't
   *    compete visually with the main content. Lets the page have a
   *    rhythm of "page header → control strip → main card → main card"
   *    instead of "card → card → card → card."
   */
  variant?: CardVariant;
}

/**
 * Reusable section container. Two variants — see `variant` doc above.
 *
 * Mirrors Pugio's card aesthetic but tuned to Night Watch: subtle border,
 * no heavy shadow, typographic title. Quiet variant deliberately strips
 * everything except the title row + content gap, so callers can place
 * controls inline with the page flow.
 */
export function Card({
  title,
  trailing,
  children,
  paddingProp,
  variant = 'solid',
}: CardProps) {
  const isQuiet = variant === 'quiet';
  const padding = paddingProp ?? (isQuiet ? '$0' : '$4');

  return (
    <YStack
      backgroundColor={isQuiet ? 'transparent' : '$color2'}
      borderWidth={isQuiet ? 0 : 1}
      borderColor="$borderColor"
      borderRadius={isQuiet ? '$0' : '$3'}
      padding={padding as any}
      gap="$3"
    >
      {(title || trailing) && (
        <XStack justifyContent="space-between" alignItems="center">
          {title ? (
            <Text
              fontSize={isQuiet ? 11 : 13}
              color={isQuiet ? '$color9' : '$color11'}
              fontWeight="600"
              letterSpacing={isQuiet ? 0.5 : 0.3}
              textTransform={isQuiet ? 'uppercase' : 'none'}
            >
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
