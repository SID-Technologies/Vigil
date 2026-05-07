import { useEffect, useState } from 'react';
import { NavLink, useLocation } from 'react-router-dom';
import {
  ArrowCircleUp,
  ChartLine,
  Clock,
  Gauge,
  Gear,
  Lightning,
  Target as TargetIcon,
} from '@phosphor-icons/react';
import { getVersion } from '@tauri-apps/api/app';
import { XStack, YStack, Text } from 'tamagui';

import { useAccent, useThemeController } from '@repo/configs/themeController';

import { useUpdater } from '../hooks/useUpdater';

interface NavItem {
  to: string;
  icon: React.ComponentType<{ size?: number; weight?: 'regular' | 'fill'; color?: string }>;
  label: string;
  /** Phase 5 routes show a "soon" tag and visually deemphasized state. */
  soon?: boolean;
}

const NAV_ITEMS: NavItem[] = [
  { to: '/', icon: Gauge, label: 'Dashboard' },
  { to: '/history', icon: ChartLine, label: 'History' },
  { to: '/outages', icon: Lightning, label: 'Outages' },
  { to: '/targets', icon: TargetIcon, label: 'Targets' },
  { to: '/settings', icon: Gear, label: 'Settings' },
];

export function Sidebar() {
  const location = useLocation();
  const { isDark, toggleTheme } = useThemeController();
  const accent = useAccent();
  const { available, installing, install } = useUpdater();
  const [version, setVersion] = useState('dev');

  useEffect(() => {
    getVersion().then(setVersion).catch(() => setVersion('dev'));
  }, []);

  return (
    <YStack
      width={220}
      backgroundColor="$color2"
      borderRightWidth={1}
      borderRightColor="$borderColor"
      paddingVertical="$3"
      justifyContent="space-between"
    >
      <YStack gap="$1">
        <XStack paddingHorizontal="$3" paddingBottom="$3" alignItems="baseline" gap="$2">
          <Text fontSize={20} fontWeight="700" color="$color12" fontFamily="$heading">
            Vigil
          </Text>
          <Text fontSize={11} color="$color8">night watch</Text>
        </XStack>

        <YStack gap="$0.5" paddingHorizontal="$2">
          {NAV_ITEMS.map(({ to, icon: Icon, label, soon }) => {
            const isActive = to === '/' ? location.pathname === '/' : location.pathname.startsWith(to);
            return (
              <NavLink
                key={to}
                to={to}
                end={to === '/'}
                style={{ textDecoration: 'none', opacity: soon ? 0.55 : 1 }}
              >
                <XStack
                  paddingHorizontal="$2.5"
                  paddingVertical="$2"
                  borderRadius="$2"
                  gap="$2"
                  alignItems="center"
                  justifyContent="space-between"
                  backgroundColor={isActive ? '$color4' : 'transparent'}
                  hoverStyle={{ backgroundColor: isActive ? '$color4' : '$color3' }}
                  cursor="pointer"
                >
                  <XStack gap="$2" alignItems="center">
                    <Icon
                      size={16}
                      weight={isActive ? 'fill' : 'regular'}
                      color={isActive ? 'var(--color12)' : 'var(--color9)'}
                    />
                    <Text
                      fontSize={13}
                      fontWeight={isActive ? '600' : '400'}
                      color={isActive ? '$color12' : '$color9'}
                    >
                      {label}
                    </Text>
                  </XStack>
                  {soon ? (
                    <Text fontSize={9} color="$color8" letterSpacing={0.5}>
                      SOON
                    </Text>
                  ) : null}
                </XStack>
              </NavLink>
            );
          })}
        </YStack>
      </YStack>

      <YStack paddingHorizontal="$3" gap="$2">
        <XStack
          paddingVertical="$1.5"
          paddingHorizontal="$2"
          borderRadius="$2"
          backgroundColor="$color3"
          cursor="pointer"
          alignItems="center"
          justifyContent="space-between"
          hoverStyle={{ backgroundColor: '$color4' }}
          onPress={() => toggleTheme()}
        >
          <Text fontSize={11} color="$color10">
            {isDark ? '◐ Dark' : '◑ Light'}
          </Text>
          <Text fontSize={10} color="$color8">
            click to flip
          </Text>
        </XStack>

        {available ? (
          // Update available — single row anchoring on the version number.
          // Click triggers download + install + relaunch via useUpdater.
          <XStack
            paddingVertical="$1.5"
            paddingHorizontal="$2"
            borderRadius="$2"
            backgroundColor="$accentBackground"
            borderWidth={1}
            borderColor="$accentBackground"
            alignItems="center"
            justifyContent="center"
            gap="$1.5"
            cursor={installing ? 'progress' : 'pointer'}
            opacity={installing ? 0.7 : 1}
            hoverStyle={{ opacity: installing ? 0.7 : 0.9 }}
            onPress={() => {
              if (!installing) install();
            }}
          >
            <ArrowCircleUp size={12} color={accent} weight="fill" />
            <Text fontSize={10} color="$accentColor" fontWeight="600" className="vigil-num">
              {installing
                ? `Installing v${available.version}...`
                : `v${available.currentVersion} → v${available.version}`}
            </Text>
          </XStack>
        ) : (
          <Text fontSize={10} color="$color8" textAlign="center" className="vigil-num">
            v{version}
          </Text>
        )}
      </YStack>
    </YStack>
  );
}
