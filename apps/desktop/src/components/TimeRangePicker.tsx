import { useEffect, useState } from 'react';
import { Calendar } from '@phosphor-icons/react';
import { Button, Input, XStack, YStack, Text } from 'tamagui';

const PRESETS = [
  { label: '1h', ms: 60 * 60 * 1000 },
  { label: '6h', ms: 6 * 60 * 60 * 1000 },
  { label: '24h', ms: 24 * 60 * 60 * 1000 },
  { label: '7d', ms: 7 * 24 * 60 * 60 * 1000 },
  { label: '30d', ms: 30 * 24 * 60 * 60 * 1000 },
] as const;

export type TimeRange = { fromMs: number; toMs: number };

interface TimeRangePickerProps {
  value: TimeRange;
  onChange: (range: TimeRange) => void;
}

/**
 * Range selector with five presets + a custom datetime mode.
 *
 * Internal state tracks which preset (if any) is currently active by
 * comparing toMs to "now" and (toMs - fromMs) to known preset durations.
 * When toMs is significantly stale or fromMs/toMs don't match any preset,
 * the picker shows in custom mode.
 *
 * Custom mode reveals two `<input type="datetime-local">` fields and an
 * Apply button. Apply only fires onChange after both fields parse — no
 * partial / invalid intermediate values escape the component.
 */
export function TimeRangePicker({ value, onChange }: TimeRangePickerProps) {
  const matchedPreset = matchPreset(value);
  const [customOpen, setCustomOpen] = useState<boolean>(matchedPreset == null);

  // Local draft state for the custom inputs. We don't push every keystroke
  // to onChange — only on Apply — so partial invalid values never reach
  // the chart query layer.
  const [fromDraft, setFromDraft] = useState<string>(toLocalDatetimeString(value.fromMs));
  const [toDraft, setToDraft] = useState<string>(toLocalDatetimeString(value.toMs));
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setFromDraft(toLocalDatetimeString(value.fromMs));
    setToDraft(toLocalDatetimeString(value.toMs));
  }, [value.fromMs, value.toMs]);

  const choosePreset = (presetMs: number) => {
    const now = Date.now();
    onChange({ fromMs: now - presetMs, toMs: now });
    setCustomOpen(false);
    setError(null);
  };

  const applyCustom = () => {
    const f = Date.parse(fromDraft);
    const t = Date.parse(toDraft);
    if (!Number.isFinite(f) || !Number.isFinite(t)) {
      setError('Both dates must be valid.');
      return;
    }
    if (t <= f) {
      setError('End must be after start.');
      return;
    }
    setError(null);
    onChange({ fromMs: f, toMs: t });
  };

  return (
    <YStack gap="$2">
      <XStack gap="$1" alignItems="center" flexWrap="wrap">
        {PRESETS.map((p) => {
          const active = matchedPreset === p.label && !customOpen;
          return (
            <ChipButton
              key={p.label}
              label={p.label}
              active={active}
              onPress={() => choosePreset(p.ms)}
            />
          );
        })}
        <ChipButton
          label="Custom"
          icon={<Calendar size={11} color="var(--color9)" />}
          active={customOpen}
          onPress={() => setCustomOpen((o) => !o)}
        />
      </XStack>

      {customOpen ? (
        <XStack
          gap="$2"
          alignItems="flex-end"
          padding="$2"
          backgroundColor="$color3"
          borderRadius="$2"
          borderWidth={1}
          borderColor="$borderColor"
          flexWrap="wrap"
        >
          <YStack gap="$1">
            <Text fontSize={9} color="$color8" letterSpacing={0.5} fontWeight="600">
              FROM
            </Text>
            <Input
              size="$2"
              width={195}
              // @ts-expect-error — Tamagui's Input passes type to underlying <input>
              type="datetime-local"
              value={fromDraft}
              onChangeText={setFromDraft}
            />
          </YStack>
          <YStack gap="$1">
            <Text fontSize={9} color="$color8" letterSpacing={0.5} fontWeight="600">
              TO
            </Text>
            <Input
              size="$2"
              width={195}
              // @ts-expect-error — Tamagui's Input passes type to underlying <input>
              type="datetime-local"
              value={toDraft}
              onChangeText={setToDraft}
            />
          </YStack>
          <Button
            size="$2"
            backgroundColor="$accentBackground"
            color="$accentColor"
            onPress={applyCustom}
          >
            Apply
          </Button>
          {error ? (
            <Text fontSize={10} color="$red10" alignSelf="center">
              {error}
            </Text>
          ) : null}
        </XStack>
      ) : null}
    </YStack>
  );
}

// ============================================================================
// Helpers
// ============================================================================

function toLocalDatetimeString(ms: number): string {
  const d = new Date(ms);
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function matchPreset(value: TimeRange): string | null {
  const now = Date.now();
  if (Math.abs(value.toMs - now) > 30_000) return null;
  const dur = value.toMs - value.fromMs;
  for (const p of PRESETS) {
    if (Math.abs(dur - p.ms) < 60_000) return p.label;
  }
  return null;
}

/**
 * Convenience: standard "last 24h" range for callers needing an initial
 * value at component mount.
 */
export function defaultRange(presetMs = 24 * 60 * 60 * 1000): TimeRange {
  const now = Date.now();
  return { fromMs: now - presetMs, toMs: now };
}

function ChipButton({
  label,
  icon,
  active,
  onPress,
}: {
  label: string;
  icon?: React.ReactNode;
  active: boolean;
  onPress: () => void;
}) {
  return (
    <XStack
      paddingHorizontal="$2.5"
      paddingVertical="$1.5"
      gap="$1.5"
      alignItems="center"
      borderRadius="$1.5"
      borderWidth={1}
      borderColor={active ? '$accentBackground' : '$borderColor'}
      backgroundColor={active ? '$accentBackground' : 'transparent'}
      cursor="pointer"
      hoverStyle={{ backgroundColor: active ? '$accentBackground' : '$color3' }}
      pressStyle={{ opacity: 0.85 }}
      animation="quick"
      onPress={onPress}
    >
      {icon}
      <Text
        fontSize={11}
        fontWeight={active ? '600' : '500'}
        color={active ? '$accentColor' : '$color11'}
      >
        {label}
      </Text>
    </XStack>
  );
}
