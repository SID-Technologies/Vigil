import { Button, XStack, YStack, Text } from 'tamagui';

import { useUpdater } from '../hooks/useUpdater';

export function UpdateBanner() {
  const { available, installing, install, dismiss } = useUpdater();

  if (!available) return null;

  return (
    <YStack
      position="absolute"
      bottom="$3"
      right="$3"
      maxWidth={340}
      padding="$3"
      gap="$2"
      borderRadius="$3"
      backgroundColor="$color2"
      borderWidth={1}
      borderColor="$borderColor"
      shadowColor="$shadowColor"
      shadowOffset={{ width: 0, height: 4 }}
      shadowRadius={12}
      shadowOpacity={0.25}
      zIndex={1000}
    >
      <XStack gap="$2" alignItems="baseline">
        <Text fontSize={13} color="$color12" fontWeight="600">
          Update available
        </Text>
        <Text fontSize={11} color="$color9" className="vigil-num">
          v{available.currentVersion} → v{available.version}
        </Text>
      </XStack>
      <Text fontSize={11} color="$color10">
        Vigil will download the new version and restart.
      </Text>
      <XStack gap="$2" justifyContent="flex-end">
        <Button size="$2" chromeless onPress={dismiss} disabled={installing}>
          Later
        </Button>
        <Button size="$2" theme="accent" onPress={install} disabled={installing}>
          {installing ? 'Installing…' : 'Restart & install'}
        </Button>
      </XStack>
    </YStack>
  );
}
