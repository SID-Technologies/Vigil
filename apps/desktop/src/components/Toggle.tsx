import { XStack, YStack } from 'tamagui';

interface ToggleProps {
  checked: boolean;
  onCheckedChange: (next: boolean) => void;
  size?: 'sm' | 'md';
  disabled?: boolean;
}

/**
 * Custom toggle. We use this instead of Tamagui's Switch because the
 * 2.0.0-rc series has flaky checked-state rendering — the thumb
 * sometimes doesn't track the checked prop after the first render, and
 * passing backgroundColor directly fights internal styling.
 *
 * This implementation is ~30 lines, deterministic, and uses the same
 * theme tokens so it visually matches the rest of the app. When/if
 * Tamagui's Switch stabilizes we can swap back, but the surface area
 * is small enough that staying off it is fine.
 */
export function Toggle({ checked, onCheckedChange, size = 'sm', disabled }: ToggleProps) {
  const w = size === 'sm' ? 32 : 44;
  const h = size === 'sm' ? 18 : 24;
  const pad = 2;
  const thumb = h - pad * 2;
  const thumbX = checked ? w - thumb - pad : pad;

  return (
    <XStack
      // Native <button> semantics so the toggle is reachable via Tab and
      // toggles on Space/Enter. tag="button" makes Tamagui render the
      // underlying <button> on web; role+aria for accessibility.
      tag="button"
      role="switch"
      aria-checked={checked}
      aria-disabled={disabled}
      width={w}
      height={h}
      borderRadius={999}
      backgroundColor={checked ? '$accentBackground' : '$color5'}
      borderWidth={1}
      borderColor={checked ? '$accentBackground' : '$color7'}
      cursor={disabled ? 'not-allowed' : 'pointer'}
      opacity={disabled ? 0.5 : 1}
      alignItems="center"
      animation="quick"
      onPress={(e: any) => {
        e?.stopPropagation?.();
        if (!disabled) onCheckedChange(!checked);
      }}
      hoverStyle={disabled ? undefined : { borderColor: checked ? '$accentBackground' : '$color8' }}
      pressStyle={disabled ? undefined : { opacity: 0.85 }}
      // Keyboard focus ring is handled globally by the :focus-visible
      // rule in styles/global.css (matches role="switch").
    >
      <YStack
        position="absolute"
        top={pad - 1}
        x={thumbX}
        width={thumb}
        height={thumb}
        borderRadius={999}
        backgroundColor={checked ? '$accentColor' : '$color12'}
        animation="quick"
      />
    </XStack>
  );
}
