import type { ReactNode } from 'react';
import { Dialog, Unspaced, XStack, YStack, Text, Button } from 'tamagui';
import { X } from '@phosphor-icons/react';

interface ModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description?: string;
  children: ReactNode;
  /** Optional footer slot — typically Cancel + Save buttons. */
  footer?: ReactNode;
  /** Modal width in px. Default 480. */
  width?: number;
}

/**
 * Lightweight wrapper around Tamagui's Dialog primitive. Centralizes the
 * styling so every modal in the app feels the same. Built on Dialog rather
 * than Sheet because dialogs are better for forms (small, focused, easy to
 * dismiss) and Sheet is better for bottom-up panels.
 */
export function Modal({
  open,
  onOpenChange,
  title,
  description,
  children,
  footer,
  width = 480,
}: ModalProps) {
  return (
    <Dialog modal open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Overlay
          key="overlay"
          animation="quick"
          opacity={0.6}
          backgroundColor="$color1"
          enterStyle={{ opacity: 0 }}
          exitStyle={{ opacity: 0 }}
        />
        <Dialog.Content
          key="content"
          backgroundColor="$color2"
          borderColor="$borderColor"
          borderWidth={1}
          borderRadius="$3"
          padding="$4"
          width={width}
          gap="$3"
          animation="quick"
          enterStyle={{ y: -8, opacity: 0 }}
          exitStyle={{ y: -8, opacity: 0 }}
        >
          <XStack justifyContent="space-between" alignItems="flex-start">
            <YStack flex={1} gap="$1">
              <Dialog.Title>
                <Text fontSize={16} fontWeight="700" color="$color12" fontFamily="$heading">
                  {title}
                </Text>
              </Dialog.Title>
              {description ? (
                <Dialog.Description>
                  <Text fontSize={12} color="$color9">
                    {description}
                  </Text>
                </Dialog.Description>
              ) : null}
            </YStack>
            <Unspaced>
              <Dialog.Close asChild>
                <Button
                  size="$2"
                  circular
                  chromeless
                  icon={<X size={14} color="var(--color9)" />}
                />
              </Dialog.Close>
            </Unspaced>
          </XStack>
          {children}
          {footer ? (
            <XStack justifyContent="flex-end" gap="$2">
              {footer}
            </XStack>
          ) : null}
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog>
  );
}
