import type { ReactNode } from 'react';
import { YStack, Text } from 'tamagui';

interface FormFieldProps {
  label: string;
  /** Optional help text shown below the input. */
  hint?: string;
  /** Optional inline error — turns the wrapper red. */
  error?: string;
  children: ReactNode;
}

/**
 * Labeled input wrapper for forms. Consistent label/hint typography across
 * Settings and the Add-Target modal.
 */
export function FormField({ label, hint, error, children }: FormFieldProps) {
  return (
    <YStack gap="$1.5">
      <Text fontSize={11} color="$color10" letterSpacing={0.4} fontWeight="600">
        {label.toUpperCase()}
      </Text>
      {children}
      {error ? (
        <Text fontSize={11} color="$red10">
          {error}
        </Text>
      ) : hint ? (
        <Text fontSize={11} color="$color8">
          {hint}
        </Text>
      ) : null}
    </YStack>
  );
}
