import { createContext, useContext, useState, useEffect, useCallback, type FC, ReactNode } from 'react';

// Base theme (light/dark)
export type ThemeMode = 'light' | 'dark' | 'system';

// Vigil ships a single theme — `nightwatch`. The style enum and picker were
// removed in v0.0.x because the alternates (default/torch/statio) were never
// brought to parity with nightwatch's contrast and the chart palette assumes
// the watchfire-amber accent. Type kept for backwards-compatible callers.
export type ThemeStyle = 'nightwatch';

// Accent color options (legacy — only nightwatch's amber is used now)
export type AccentColor = 'orange';

// The actual Tamagui theme name
export type TamaguiThemeName = 'light_nightwatch' | 'dark_nightwatch';

// Resolved accent color hex per mode. Used directly (not via CSS var) because
// Tauri's WebView doesn't reliably propagate document-root CSS variables into
// recharts SVG strokes or phosphor icon `color` props on first paint.
const ACCENT_DARK = '#e0a458'; // watchfire amber against #0b1116 slate
const ACCENT_LIGHT = '#b8742a'; // darker burnt-amber against #f6f1e7 tan

interface ThemeContextType {
  themeMode: ThemeMode;
  themeStyle: ThemeStyle;
  accentColor: AccentColor;
  resolvedTheme: TamaguiThemeName;
  isDark: boolean;
  setThemeMode: (mode: ThemeMode) => void;
  toggleTheme: () => void;
  isTransitioning: boolean;
}

const ThemeContext = createContext<ThemeContextType>({
  themeMode: 'dark',
  themeStyle: 'nightwatch',
  accentColor: 'orange',
  resolvedTheme: 'dark_nightwatch',
  isDark: true,
  setThemeMode: () => {},
  toggleTheme: () => {},
  isTransitioning: false,
});

const getSystemTheme = (): 'light' | 'dark' => {
  if (typeof window !== 'undefined' && window.matchMedia) {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  }
  return 'dark';
};

const LS_THEME_MODE = 'vigil-theme-mode';

export const ThemeProvider: FC<{ children: ReactNode }> = ({ children }) => {
  const [themeMode, setThemeModeState] = useState<ThemeMode>(() => {
    try {
      const stored = localStorage.getItem(LS_THEME_MODE);
      if (stored === 'light' || stored === 'dark' || stored === 'system') return stored;
    } catch {}
    return 'dark';
  });
  const [systemTheme, setSystemTheme] = useState<'light' | 'dark'>(getSystemTheme);
  const [isTransitioning, setIsTransitioning] = useState(false);

  useEffect(() => {
    if (typeof window === 'undefined' || !window.matchMedia) return;

    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    const handleChange = (e: MediaQueryListEvent) => {
      setSystemTheme(e.matches ? 'dark' : 'light');
    };

    mediaQuery.addEventListener('change', handleChange);
    return () => mediaQuery.removeEventListener('change', handleChange);
  }, []);

  const isDark = themeMode === 'system' ? systemTheme === 'dark' : themeMode === 'dark';
  const resolvedTheme: TamaguiThemeName = isDark ? 'dark_nightwatch' : 'light_nightwatch';

  // Mirror the resolved accent into a CSS variable for consumers that work
  // through CSS (global.css uses it for focus rings + selection highlight).
  // React-tree consumers should prefer the useAccent() hook below — it reads
  // the same value but doesn't depend on CSS var propagation, which is flaky
  // in Tauri's WebView for inline styles + recharts SVG strokes.
  useEffect(() => {
    const accent = isDark ? ACCENT_DARK : ACCENT_LIGHT;
    const contrast = isDark ? '#0b1116' : '#ffffff';
    const root = document.documentElement;
    root.style.setProperty('--accentColor', accent);
    root.style.setProperty('--accentColorContrast', contrast);
  }, [isDark]);

  const setThemeMode = useCallback((mode: ThemeMode) => {
    setIsTransitioning(true);
    setTimeout(() => {
      setThemeModeState(mode);
      try { localStorage.setItem(LS_THEME_MODE, mode); } catch {}
    }, 0);
    setTimeout(() => {
      setIsTransitioning(false);
    }, 300);
  }, []);

  const toggleTheme = useCallback(() => {
    setThemeMode(isDark ? 'light' : 'dark');
  }, [isDark, setThemeMode]);

  return (
    <ThemeContext.Provider
      value={{
        themeMode,
        themeStyle: 'nightwatch',
        accentColor: 'orange',
        resolvedTheme,
        isDark,
        setThemeMode,
        toggleTheme,
        isTransitioning,
      }}
    >
      {children}
    </ThemeContext.Provider>
  );
};

export const useThemeController = () => useContext(ThemeContext);

// useAccent returns the resolved accent hex string for the current mode.
// Use this anywhere you'd otherwise reach for `var(--accentColor)` in an
// inline style, recharts prop, or phosphor icon color — those don't always
// pick up the CSS variable in Tauri's WebView, and React state is reliable.
export const useAccent = (): string => {
  const { isDark } = useThemeController();
  return isDark ? ACCENT_DARK : ACCENT_LIGHT;
};

// Legacy export for backwards compatibility
export const useTheme = useThemeController;
