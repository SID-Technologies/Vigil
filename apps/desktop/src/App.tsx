import { BrowserRouter, Route, Routes } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

// @ts-expect-error — CSS side-effect import
import '@fontsource/inter';
// @ts-expect-error — CSS side-effect import
import '@fontsource/space-grotesk';

import { config as tamaguiConfig } from '@repo/configs/tamagui.config';
import { ThemeProvider, useThemeController } from '@repo/configs/themeController';
import { createTamagui, TamaguiProvider, Theme, XStack, YStack } from 'tamagui';

import { Sidebar } from './components/Sidebar';
import { WelcomeTour } from './components/WelcomeTour';
import { useMenuEvents } from './hooks/useMenuEvents';
import { DashboardPage } from './pages/Dashboard';
import { HistoryPage } from './pages/History';
import { OutagesPage } from './pages/Outages';
import { SettingsPage } from './pages/Settings';
import { TargetsPage } from './pages/Targets';

const tamaguiCreatedConfig = createTamagui(tamaguiConfig);

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // Bumped to 60s so route navigation doesn't re-fetch the same data
      // immediately on mount. Live components keep their own
      // refetchInterval, so this only affects background staleness.
      staleTime: 60_000,
      // Cache stays around for 10 min after the last subscriber unmounts —
      // means navigating away from a page and back keeps its data warm.
      gcTime: 10 * 60_000,
      retry: 1,
      refetchOnWindowFocus: false,
      refetchOnMount: false,
    },
  },
});

function AppContent() {
  const { resolvedTheme } = useThemeController();
  // Mount the menu-event listener once at the App level. New Report
  // navigation goes through the History page, which detects ?report=1
  // on its own — App-level handler doesn't need to know about it.
  useMenuEvents();

  return (
    <TamaguiProvider config={tamaguiCreatedConfig} defaultTheme="dark">
      <Theme name={resolvedTheme}>
        <XStack flex={1} minHeight="100vh" backgroundColor="$background">
          <Sidebar />
          <YStack flex={1} overflow="scroll">
            <Routes>
              <Route path="/" element={<DashboardPage />} />
              <Route path="/history" element={<HistoryPage />} />
              <Route path="/outages" element={<OutagesPage />} />
              <Route path="/targets" element={<TargetsPage />} />
              <Route path="/settings" element={<SettingsPage />} />
            </Routes>
          </YStack>
        </XStack>
        <WelcomeTour />
      </Theme>
    </TamaguiProvider>
  );
}

export function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <BrowserRouter>
          <AppContent />
        </BrowserRouter>
      </ThemeProvider>
    </QueryClientProvider>
  );
}
