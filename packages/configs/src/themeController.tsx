import { createContext, useContext, useState, useEffect, useCallback, type FC, ReactNode } from 'react';

// Base theme (light/dark)
export type ThemeMode = 'light' | 'dark' | 'system';

// Theme style options. Vigil ships `nightwatch` as the default; the others are
// inherited from the Pugio config so users can switch.
export type ThemeStyle = 'nightwatch' | 'default' | 'torch' | 'retro' | 'odyssey';

// Accent color options (for default theme only)
export type AccentColor = 'blue' | 'green' | 'purple' | 'orange' | 'pink' | 'teal';

// The actual Tamagui theme name
export type TamaguiThemeName =
  | 'light' | 'dark'
  | 'light_blue' | 'dark_blue'
  | 'light_green' | 'dark_green'
  | 'light_torch' | 'dark_torch'
  | 'light_retro' | 'dark_retro'
  | 'light_odyssey' | 'dark_odyssey'
  | 'light_nightwatch' | 'dark_nightwatch';

interface ThemeContextType {
  themeMode: ThemeMode;
  themeStyle: ThemeStyle;
  accentColor: AccentColor;
  resolvedTheme: TamaguiThemeName;
  isDark: boolean;
  isRetro: boolean;
  setThemeMode: (mode: ThemeMode) => void;
  setThemeStyle: (style: ThemeStyle) => void;
  setAccentColor: (color: AccentColor) => void;
  toggleTheme: () => void;
  isTransitioning: boolean;
}

const ThemeContext = createContext<ThemeContextType>({
  themeMode: 'dark',
  themeStyle: 'nightwatch',
  accentColor: 'blue',
  resolvedTheme: 'dark_nightwatch',
  isDark: true,
  isRetro: false,
  setThemeMode: () => {},
  setThemeStyle: () => {},
  setAccentColor: () => {},
  toggleTheme: () => {},
  isTransitioning: false,
});

// Helper to detect system theme preference
const getSystemTheme = (): 'light' | 'dark' => {
  if (typeof window !== 'undefined' && window.matchMedia) {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  }
  return 'dark';
};

// Helper to resolve theme name for Tamagui
const resolveThemeName = (
  mode: ThemeMode,
  style: ThemeStyle,
  systemTheme: 'light' | 'dark'
): TamaguiThemeName => {
  const baseTheme = mode === 'system' ? systemTheme : mode;

  // Named theme styles map to sub-themes
  if (style !== 'default') {
    return `${baseTheme}_${style}` as TamaguiThemeName;
  }

  // Default theme
  return baseTheme;
};

const LS_THEME_MODE = 'vigil-theme-mode';
const LS_THEME_STYLE = 'vigil-theme-style';

const VALID_STYLES: ThemeStyle[] = ['nightwatch', 'default', 'torch', 'retro', 'odyssey'];

export const ThemeProvider: FC<{ children: ReactNode }> = ({ children }) => {
  const [themeMode, setThemeModeState] = useState<ThemeMode>(() => {
    try {
      const stored = localStorage.getItem(LS_THEME_MODE);
      if (stored === 'light' || stored === 'dark' || stored === 'system') return stored;
    } catch {}
    // Vigil defaults to dark — the watchman lives at night.
    return 'dark';
  });
  const [themeStyle, setThemeStyleState] = useState<ThemeStyle>(() => {
    try {
      const stored = localStorage.getItem(LS_THEME_STYLE);
      if (stored && VALID_STYLES.includes(stored as ThemeStyle)) return stored as ThemeStyle;
    } catch {}
    return 'nightwatch';
  });
  const [accentColor, setAccentColorState] = useState<AccentColor>('orange');
  const [systemTheme, setSystemTheme] = useState<'light' | 'dark'>(getSystemTheme);
  const [isTransitioning, setIsTransitioning] = useState(false);

  // Listen for system theme changes
  useEffect(() => {
    if (typeof window === 'undefined' || !window.matchMedia) return;

    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    const handleChange = (e: MediaQueryListEvent) => {
      setSystemTheme(e.matches ? 'dark' : 'light');
    };

    mediaQuery.addEventListener('change', handleChange);
    return () => mediaQuery.removeEventListener('change', handleChange);
  }, []);

  // Resolve the actual theme
  const isDark = themeMode === 'system' ? systemTheme === 'dark' : themeMode === 'dark';
  const isRetro = themeStyle === 'retro';
  const resolvedTheme = resolveThemeName(themeMode, themeStyle, systemTheme);

  // Per-style accent color as CSS custom property.
  // Tamagui's createThemes doesn't propagate `extra` to child themes, so we
  // set the accent color manually on the document root so components can use
  // `var(--accentColor)` for things like chart highlights and active nav state.
  useEffect(() => {
    const accentMap: Record<ThemeStyle, { light: string; dark: string }> = {
      nightwatch: { light: '#b8742a', dark: '#e0a458' },
      default: { light: '#0090ff', dark: '#0090ff' },
      torch: { light: '#8b5cf6', dark: '#8b5cf6' },
      retro: { light: '#d97706', dark: '#f59e0b' },
      odyssey: { light: '#b08d3e', dark: '#c8a84e' },
    };
    const colors = accentMap[themeStyle] || accentMap.nightwatch;
    const root = document.documentElement;
    root.style.setProperty('--accentColor', isDark ? colors.dark : colors.light);
    root.style.setProperty('--accentColorContrast', isDark ? '#0b1116' : '#ffffff');
  }, [themeStyle, isDark]);

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

  const setThemeStyle = useCallback((style: ThemeStyle) => {
    setIsTransitioning(true);
    setTimeout(() => {
      setThemeStyleState(style);
      try { localStorage.setItem(LS_THEME_STYLE, style); } catch {}
    }, 0);
    setTimeout(() => {
      setIsTransitioning(false);
    }, 300);
  }, []);

  const setAccentColor = useCallback((color: AccentColor) => {
    setIsTransitioning(true);
    setTimeout(() => {
      setAccentColorState(color);
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
        themeStyle,
        accentColor,
        resolvedTheme,
        isDark,
        isRetro,
        setThemeMode,
        setThemeStyle,
        setAccentColor,
        toggleTheme,
        isTransitioning,
      }}
    >
      {children}
    </ThemeContext.Provider>
  );
};

export const useThemeController = () => useContext(ThemeContext);

// Legacy export for backwards compatibility
export const useTheme = useThemeController;
