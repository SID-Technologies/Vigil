import { useEffect, useState } from 'react';
import { Keyboard } from '@phosphor-icons/react';
import { Dialog, Unspaced, XStack, YStack, Text, Button } from 'tamagui';

const IS_MAC =
  typeof navigator !== 'undefined' && /Mac|iPod|iPhone|iPad/.test(navigator.platform);
const MOD = IS_MAC ? '⌘' : 'Ctrl';

interface Shortcut {
  keys: string[];
  description: string;
}

interface ShortcutGroup {
  title: string;
  items: Shortcut[];
}

// Source of truth for which shortcuts the app supports. If you wire a new
// keyboard handler somewhere, list it here too — the overlay is the user-
// facing index.
const GROUPS: ShortcutGroup[] = [
  {
    title: 'Navigation',
    items: [
      { keys: [`${MOD}`, '1'], description: 'Dashboard' },
      { keys: [`${MOD}`, '2'], description: 'History' },
      { keys: [`${MOD}`, '3'], description: 'Outages' },
      { keys: [`${MOD}`, '4'], description: 'Targets' },
      { keys: [`${MOD}`, '5'], description: 'Settings' },
    ],
  },
  {
    title: 'Actions',
    items: [
      { keys: [`${MOD}`, 'S'], description: 'Save Settings (when on Settings page)' },
      { keys: ['Esc'], description: 'Close modal or dialog' },
      { keys: ['Tab'], description: 'Move focus to the next control' },
      { keys: ['Space', '/', 'Enter'], description: 'Toggle the focused switch / button' },
    ],
  },
  {
    title: 'Help',
    items: [
      { keys: ['Shift', '?'], description: 'Show this overlay' },
    ],
  },
];

/**
 * KeyboardShortcuts — Shift+? overlay listing every shortcut Vigil supports.
 *
 * Mounted at App level so the listener works on every page. The convention
 * matches Linear, GitHub, Slack — Shift+? brings up a cheat sheet, Esc
 * closes it. Discoverable for power users without cluttering the UI for
 * everyone else.
 *
 * Also exposes `useKeyboardShortcutsControl()` so a UI button (e.g. in
 * Settings → Help) can open it without needing a key press.
 */
export function KeyboardShortcuts() {
  const [open, setOpen] = useState(false);

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      // Don't grab the shortcut if the user is typing in an input.
      const target = e.target as HTMLElement | null;
      if (target && /^(INPUT|TEXTAREA|SELECT)$/.test(target.tagName)) return;
      if (target?.isContentEditable) return;

      if (e.key === '?' && (e.shiftKey || e.code === 'Slash')) {
        e.preventDefault();
        setOpen((o) => !o);
      } else if (e.key === 'Escape' && open) {
        setOpen(false);
      }
    };
    window.addEventListener('keydown', handler);
    // Listen for a custom event so other components (Settings Help row)
    // can ask us to open without coupling state across the tree.
    const opener = () => setOpen(true);
    window.addEventListener('vigil:open-shortcuts', opener);
    return () => {
      window.removeEventListener('keydown', handler);
      window.removeEventListener('vigil:open-shortcuts', opener);
    };
  }, [open]);

  return (
    <Dialog modal open={open} onOpenChange={setOpen}>
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
          padding="$5"
          width={520}
          gap="$4"
          animation="quick"
          enterStyle={{ y: -8, opacity: 0 }}
          exitStyle={{ y: -8, opacity: 0 }}
        >
          <XStack alignItems="center" gap="$2.5">
            <YStack
              width={36}
              height={36}
              borderRadius="$2"
              backgroundColor="$color3"
              borderWidth={1}
              borderColor="$borderColor"
              alignItems="center"
              justifyContent="center"
            >
              <Keyboard size={18} color="var(--color11)" weight="duotone" />
            </YStack>
            <YStack flex={1}>
              <Dialog.Title>
                <Text fontSize={17} fontWeight="700" color="$color12" fontFamily="$heading">
                  Keyboard shortcuts
                </Text>
              </Dialog.Title>
              <Text fontSize={11} color="$color9">
                Press <Kbd>Shift</Kbd> + <Kbd>?</Kbd> any time to bring this back.
              </Text>
            </YStack>
            <Unspaced>
              <Dialog.Close asChild>
                <Button size="$2" chromeless onPress={() => setOpen(false)}>
                  <Text fontSize={11} color="$color9">Close</Text>
                </Button>
              </Dialog.Close>
            </Unspaced>
          </XStack>

          <YStack gap="$3.5">
            {GROUPS.map((g) => (
              <YStack key={g.title} gap="$1.5">
                <Text
                  fontSize={10}
                  color="$color8"
                  fontWeight="600"
                  letterSpacing={0.6}
                  textTransform="uppercase"
                >
                  {g.title}
                </Text>
                <YStack
                  borderWidth={1}
                  borderColor="$borderColor"
                  borderRadius="$2"
                  backgroundColor="$color1"
                  overflow="hidden"
                >
                  {g.items.map((s, i) => (
                    <XStack
                      key={`${g.title}-${i}`}
                      paddingVertical="$2"
                      paddingHorizontal="$3"
                      alignItems="center"
                      justifyContent="space-between"
                      gap="$2"
                      borderTopWidth={i === 0 ? 0 : 1}
                      borderTopColor="$borderColor"
                    >
                      <Text fontSize={12} color="$color11" flex={1}>
                        {s.description}
                      </Text>
                      <XStack gap="$1" alignItems="center">
                        {s.keys.map((k, j) => (
                          <XStack key={j} alignItems="center" gap="$1">
                            <Kbd>{k}</Kbd>
                            {j < s.keys.length - 1 ? (
                              <Text fontSize={10} color="$color8">
                                +
                              </Text>
                            ) : null}
                          </XStack>
                        ))}
                      </XStack>
                    </XStack>
                  ))}
                </YStack>
              </YStack>
            ))}
          </YStack>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog>
  );
}

/**
 * Inline <kbd>-style key cap. Tabular numerals so multi-digit keys
 * (rare but possible) line up; mono-ish look from $body without pulling in
 * a separate font.
 */
function Kbd({ children }: { children: React.ReactNode }) {
  return (
    <YStack
      paddingHorizontal="$1.5"
      paddingVertical={2}
      borderRadius="$1"
      backgroundColor="$color3"
      borderWidth={1}
      borderColor="$borderColor"
      borderBottomWidth={2}
      minWidth={20}
      alignItems="center"
    >
      <Text
        fontSize={10}
        color="$color12"
        fontWeight="600"
        fontFamily="$body"
        style={{ fontVariantNumeric: 'tabular-nums' }}
      >
        {children}
      </Text>
    </YStack>
  );
}

export function openShortcuts() {
  window.dispatchEvent(new Event('vigil:open-shortcuts'));
}
