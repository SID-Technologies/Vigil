import type { ReactNode } from 'react';
import { Button, YStack, Text } from 'tamagui';

import { Modal } from './Modal';

interface ConfirmDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  /** Body text. Pass JSX for richer descriptions. */
  description?: ReactNode;
  /** Label on the confirm button (e.g. "Delete", "Disable", "Apply"). */
  confirmLabel?: string;
  /** When true, the confirm button uses red styling — for destructive actions. */
  destructive?: boolean;
  /** Disabled while a mutation is in flight. */
  loading?: boolean;
  onConfirm: () => void;
  /** Optional override for the cancel button label. */
  cancelLabel?: string;
}

/**
 * Reusable Tamagui-themed confirm dialog. Replaces every native `confirm()`
 * in the app — keeps focus inside the window, matches Night Watch styling,
 * supports destructive vs non-destructive variants.
 *
 * Usage:
 *
 *   const [open, setOpen] = useState(false);
 *   <Button onPress={() => setOpen(true)}>Delete</Button>
 *   <ConfirmDialog
 *     open={open}
 *     onOpenChange={setOpen}
 *     title="Delete target?"
 *     description="This cannot be undone."
 *     destructive
 *     confirmLabel="Delete"
 *     onConfirm={() => { delete(); setOpen(false); }}
 *   />
 */
export function ConfirmDialog({
  open,
  onOpenChange,
  title,
  description,
  confirmLabel = 'Confirm',
  cancelLabel = 'Cancel',
  destructive,
  loading,
  onConfirm,
}: ConfirmDialogProps) {
  return (
    <Modal
      open={open}
      onOpenChange={onOpenChange}
      title={title}
      width={420}
      footer={
        <>
          <Button size="$3" chromeless onPress={() => onOpenChange(false)} disabled={loading}>
            {cancelLabel}
          </Button>
          <Button
            size="$3"
            backgroundColor={destructive ? '$red10' : '$accentBackground'}
            color={destructive ? '$color1' : '$accentColor'}
            onPress={onConfirm}
            disabled={loading}
            hoverStyle={{ opacity: 0.9 }}
          >
            {loading ? 'Working…' : confirmLabel}
          </Button>
        </>
      }
    >
      {description ? (
        <YStack gap="$2">
          {typeof description === 'string' ? (
            <Text fontSize={13} color="$color11" lineHeight={18}>
              {description}
            </Text>
          ) : (
            description
          )}
        </YStack>
      ) : null}
    </Modal>
  );
}
