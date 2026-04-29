import * as Colors from '@tamagui/colors'
import { createThemes } from '@tamagui/config/v4'

// =============================================================================
// THEME STYLE TOKENS
// =============================================================================
// These define per-theme design tokens beyond just colors - shadows, radii, etc.
// Components can use these for theme-specific styling variations.

const defaultStyleTokens = {
  shadowCard: '0 1px 3px rgba(0,0,0,0.12), 0 1px 2px rgba(0,0,0,0.08)',
  shadowCardHover: '0 4px 6px rgba(0,0,0,0.1), 0 2px 4px rgba(0,0,0,0.06)',
  shadowModal: '0 25px 50px rgba(0,0,0,0.25)',
  shadowButton: '0 1px 2px rgba(0,0,0,0.05)',
  radiusStyle: 'default',
}

const torchStyleTokens = {
  shadowCard: '0 2px 8px rgba(139, 92, 246, 0.08), 0 1px 3px rgba(0,0,0,0.12)',
  shadowCardHover: '0 8px 24px rgba(139, 92, 246, 0.15), 0 4px 8px rgba(0,0,0,0.1)',
  shadowModal: '0 25px 50px rgba(0,0,0,0.5)',
  shadowButton: '0 2px 4px rgba(139, 92, 246, 0.2)',
  shadowGlow: '0 0 40px rgba(139, 92, 246, 0.3)',
  radiusStyle: 'default',
}

// Night Watch — watchman's tower at 2am: cold dark slate, one warm amber light.
// Subtle shadows, sharp not soft. Engineering-grade restraint.
const nightWatchStyleTokens = {
  shadowCard: '0 1px 2px rgba(0,0,0,0.4), 0 0 0 1px rgba(255,255,255,0.02)',
  shadowCardHover: '0 4px 8px rgba(0,0,0,0.45), 0 0 0 1px rgba(224,164,88,0.08)',
  shadowModal: '0 20px 40px rgba(0,0,0,0.6)',
  shadowButton: '0 1px 2px rgba(0,0,0,0.3)',
  shadowGlow: '0 0 24px rgba(224,164,88,0.18)',
  radiusStyle: 'default',
}

// =============================================================================
// LABEL/CATEGORY COLOR PALETTE
// =============================================================================

const labelColors = {
  labelRed: '#ef4444',
  labelRose: '#f43f5e',
  labelPink: '#ec4899',
  labelOrange: '#f97316',
  labelAmber: '#f59e0b',
  labelYellow: '#eab308',
  labelLime: '#84cc16',
  labelGreen: '#22c55e',
  labelEmerald: '#10b981',
  labelTeal: '#14b8a6',
  labelCyan: '#06b6d4',
  labelSky: '#0ea5e9',
  labelBlue: '#3b82f6',
  labelIndigo: '#6366f1',
  labelViolet: '#8b5cf6',
  labelPurple: '#a855f7',
  labelFuchsia: '#d946ef',
  labelSlate: '#64748b',
}

// =============================================================================
// SEMANTIC COLOR TOKENS — default theme
// =============================================================================

const lightSemanticColors = {
  accent: Colors.blue.blue9,
  accentLight: Colors.blue.blue3,
  accentLighter: Colors.blue.blue1,
  accentDark: Colors.blue.blue10,
  accentDarker: Colors.blue.blue11,
  accentBorder: Colors.blue.blue6,
  accentBorderHover: Colors.blue.blue8,

  success: Colors.green.green10,
  successLight: Colors.green.green1,
  successLighter: Colors.green.green2,
  successDark: Colors.green.green11,
  successBorder: Colors.green.green6,

  error: Colors.red.red10,
  errorLight: Colors.red.red1,
  errorLighter: Colors.red.red2,
  errorDark: Colors.red.red11,
  errorBorder: Colors.red.red6,
  errorBorderStrong: Colors.red.red8,

  warning: Colors.yellow.yellow10,
  warningLight: Colors.yellow.yellow2,
  warningLighter: Colors.yellow.yellow1,
  warningDark: Colors.yellow.yellow11,
  warningBorder: Colors.yellow.yellow6,

  info: Colors.blue.blue10,
  infoLight: Colors.blue.blue1,
  infoDark: Colors.blue.blue11,
  infoBorder: Colors.blue.blue6,

  ...labelColors,

  textInverse: '#ffffff',
  skeleton: 'hsl(0, 0%, 90.0%)',
  skeletonLight: 'hsl(0, 0%, 94.1%)',
  overlay: 'rgba(0,0,0,0.5)',
  overlayLight: 'rgba(0,0,0,0.3)',
  transparent: 'transparent',

  borderDefault: 'hsl(0, 0%, 90.0%)',
  borderMuted: 'hsl(0, 0%, 92.0%)',
  borderStrong: 'hsl(0, 0%, 81.0%)',
  borderFocus: Colors.blue.blue7,
}

const darkSemanticColors = {
  accent: Colors.blueDark.blue9,
  accentLight: Colors.blueDark.blue3,
  accentLighter: Colors.blueDark.blue1,
  accentDark: Colors.blueDark.blue10,
  accentDarker: Colors.blueDark.blue11,
  accentBorder: Colors.blueDark.blue6,
  accentBorderHover: Colors.blueDark.blue8,

  success: Colors.greenDark.green10,
  successLight: Colors.greenDark.green1,
  successLighter: Colors.greenDark.green2,
  successDark: Colors.greenDark.green11,
  successBorder: Colors.greenDark.green6,

  error: Colors.redDark.red10,
  errorLight: Colors.redDark.red1,
  errorLighter: Colors.redDark.red2,
  errorDark: Colors.redDark.red11,
  errorBorder: Colors.redDark.red6,
  errorBorderStrong: Colors.redDark.red8,

  warning: Colors.yellowDark.yellow10,
  warningLight: Colors.yellowDark.yellow2,
  warningLighter: Colors.yellowDark.yellow1,
  warningDark: Colors.yellowDark.yellow11,
  warningBorder: Colors.yellowDark.yellow6,

  info: Colors.blueDark.blue10,
  infoLight: Colors.blueDark.blue1,
  infoDark: Colors.blueDark.blue11,
  infoBorder: Colors.blueDark.blue6,

  ...labelColors,

  textInverse: '#000000',
  skeleton: 'hsl(0, 0%, 20%)',
  skeletonLight: 'hsl(0, 0%, 15%)',
  overlay: 'rgba(0,0,0,0.7)',
  overlayLight: 'rgba(0,0,0,0.5)',
  transparent: 'transparent',

  borderDefault: 'hsl(0, 0%, 25%)',
  borderMuted: 'hsl(0, 0%, 20%)',
  borderStrong: 'hsl(0, 0%, 35%)',
  borderFocus: Colors.blueDark.blue7,
}

const darkPalette = [
  '#050505',
  '#151515',
  '#191919',
  '#232323',
  '#282828',
  '#323232',
  '#424242',
  '#494949',
  '#545454',
  '#626262',
  '#a5a5a5',
  '#fff',
]

const lightPalette = [
  '#fff',
  '#f8f8f8',
  'hsl(0, 0%, 96.3%)',
  'hsl(0, 0%, 94.1%)',
  'hsl(0, 0%, 92.0%)',
  'hsl(0, 0%, 90.0%)',
  'hsl(0, 0%, 88.5%)',
  'hsl(0, 0%, 81.0%)',
  'hsl(0, 0%, 56.1%)',
  'hsl(0, 0%, 50.3%)',
  'hsl(0, 0%, 42.5%)',
  'hsl(0, 0%, 9.0%)',
]

const lightShadows = {
  shadow1: 'rgba(0,0,0,0.04)',
  shadow2: 'rgba(0,0,0,0.08)',
  shadow3: 'rgba(0,0,0,0.16)',
  shadow4: 'rgba(0,0,0,0.24)',
  shadow5: 'rgba(0,0,0,0.32)',
  shadow6: 'rgba(0,0,0,0.4)',
}

const darkShadows = {
  shadow1: 'rgba(0,0,0,0.2)',
  shadow2: 'rgba(0,0,0,0.3)',
  shadow3: 'rgba(0,0,0,0.4)',
  shadow4: 'rgba(0,0,0,0.5)',
  shadow5: 'rgba(0,0,0,0.6)',
  shadow6: 'rgba(0,0,0,0.7)',
}

const extraColors = {
  black1: darkPalette[0],
  black2: darkPalette[1],
  black3: darkPalette[2],
  black4: darkPalette[3],
  black5: darkPalette[4],
  black6: darkPalette[5],
  black7: darkPalette[6],
  black8: darkPalette[7],
  black9: darkPalette[8],
  black10: darkPalette[9],
  black11: darkPalette[10],
  black12: darkPalette[11],
  white1: lightPalette[0],
  white2: lightPalette[1],
  white3: lightPalette[2],
  white4: lightPalette[3],
  white5: lightPalette[4],
  white6: lightPalette[5],
  white7: lightPalette[6],
  white8: lightPalette[7],
  white9: lightPalette[8],
  white10: lightPalette[9],
  white11: lightPalette[10],
  white12: lightPalette[11],
}

// =============================================================================
// TORCH THEME
// =============================================================================

const torchDarkPalette = [
  '#0a0a0b', '#111113', '#18181b', '#222225',
  '#2c2c30', '#37373d', '#47474f', '#5f5f6b',
  '#78788a', '#9898a8', '#b8b8c5', '#fafafa',
]

const torchLightPalette = [
  '#ffffff', '#fafafa', '#f5f5f7', '#ebebef',
  '#e0e0e6', '#d4d4dc', '#b8b8c5', '#9898a8',
  '#78788a', '#5f5f6b', '#37373d', '#0a0a0b',
]

const torchSemanticColorsLight = {
  accent: '#8b5cf6', accentLight: '#c4b5fd', accentLighter: '#ede9fe',
  accentDark: '#7c3aed', accentDarker: '#6d28d9',
  accentBorder: '#a78bfa', accentBorderHover: '#8b5cf6',
  accentGradientStart: '#8b5cf6', accentGradientEnd: '#a855f7',
  success: Colors.green.green10, successLight: Colors.green.green1, successBorder: Colors.green.green6,
  error: Colors.red.red10, errorLight: Colors.red.red1, errorBorder: Colors.red.red6,
  warning: Colors.yellow.yellow10, warningLight: Colors.yellow.yellow1, warningBorder: Colors.yellow.yellow6,
  ...labelColors,
  textInverse: '#ffffff',
  skeleton: '#e0e0e6',
  overlay: 'rgba(10, 10, 11, 0.5)', overlayLight: 'rgba(10, 10, 11, 0.3)',
  transparent: 'transparent',
  borderDefault: '#e0e0e6', borderMuted: '#ebebef', borderStrong: '#d4d4dc', borderFocus: '#8b5cf6',
  ...torchStyleTokens,
}

const torchSemanticColorsDark = {
  accent: '#8b5cf6', accentLight: '#2e1065', accentLighter: '#1e0a4a',
  accentDark: '#a855f7', accentDarker: '#c084fc',
  accentBorder: '#6d28d9', accentBorderHover: '#8b5cf6',
  accentGradientStart: '#8b5cf6', accentGradientEnd: '#a855f7',
  success: Colors.greenDark.green10, successLight: Colors.greenDark.green1, successBorder: Colors.greenDark.green6,
  error: Colors.redDark.red10, errorLight: Colors.redDark.red1, errorBorder: Colors.redDark.red6,
  warning: Colors.yellowDark.yellow10, warningLight: Colors.yellowDark.yellow1, warningBorder: Colors.yellowDark.yellow6,
  ...labelColors,
  textInverse: '#0a0a0b',
  skeleton: '#222225',
  overlay: 'rgba(0, 0, 0, 0.7)', overlayLight: 'rgba(0, 0, 0, 0.5)',
  transparent: 'transparent',
  borderDefault: '#27272a', borderMuted: '#222225', borderStrong: '#37373d', borderFocus: '#8b5cf6',
  ...torchStyleTokens,
}

// =============================================================================
// ODYSSEY THEME — flat theme objects merged onto base via spread
// =============================================================================

const statioDark = {
  background: '#080c12',
  backgroundHover: '#0c1018',
  backgroundPress: '#0a0e14',
  backgroundFocus: '#0c1018',
  backgroundStrong: '#05080c',
  backgroundTransparent: 'rgba(8, 12, 18, 0)',

  color: '#e8e4d8',
  colorHover: '#f4f0e4',
  colorPress: '#c4bea8',
  colorFocus: '#f4f0e4',
  colorTransparent: 'rgba(232, 228, 216, 0)',

  color1: '#080c12', color2: '#0e1420', color3: '#161e2a', color4: '#1e2836',
  color5: '#283244', color6: '#445060', color7: '#647080', color8: '#8894a4',
  color9: '#a8b4c0', color10: '#c4ccd4', color11: '#dce0e4', color12: '#e8e4d8',

  accentBackground: '#c8a84e',
  accentColor: '#080c12',

  borderColor: '#1e2836',
  borderColorHover: '#283244',
  borderColorPress: '#1e2836',
  borderColorFocus: '#c8a84e',

  shadowColor: 'rgba(4, 8, 16, 0.8)',
  shadowColorHover: 'rgba(4, 8, 16, 0.9)',
}

// =============================================================================
// NIGHT WATCH THEME — Vigil's signature theme
// =============================================================================
// Watchman's tower at 2am: cold dark slate, one warm amber light burning.
// Distinguishable from the Statio (Aegean gold) and Default (electric blue) at a glance.

const nightWatchDark = {
  // Surfaces — deep slate, slight cool undertone
  background: '#0b1116',
  backgroundHover: '#11181f',
  backgroundPress: '#0e151b',
  backgroundFocus: '#11181f',
  backgroundStrong: '#070b0f',
  backgroundTransparent: 'rgba(11, 17, 22, 0)',

  // Text — off-white with warm undertone (parchment by lantern light)
  color: '#e6edf3',
  colorHover: '#f0f6fc',
  colorPress: '#c4ccd6',
  colorFocus: '#f0f6fc',
  colorTransparent: 'rgba(230, 237, 243, 0)',

  // Neutral scale — slate with cool undertone
  color1: '#0b1116',
  color2: '#11181f',
  color3: '#161d26',
  color4: '#1c242f',
  color5: '#222b38',
  color6: '#2d3744',
  color7: '#3d4855',
  color8: '#5b6573',
  color9: '#8b949e',
  color10: '#b8c0c8',
  color11: '#d4dade',
  color12: '#e6edf3',

  // Primary accent — watchfire amber
  accentBackground: '#e0a458',
  accentColor: '#0b1116',

  // Borders — muted blue-gray
  borderColor: '#1f2933',
  borderColorHover: '#2d3744',
  borderColorPress: '#1f2933',
  borderColorFocus: '#e0a458',

  // Shadows — deep, cold
  shadowColor: 'rgba(0, 0, 0, 0.6)',
  shadowColorHover: 'rgba(0, 0, 0, 0.75)',
}

const nightWatchLight = {
  // Surfaces — warm off-white (old watchman's logbook)
  background: '#f6f1e7',
  backgroundHover: '#f0eadc',
  backgroundPress: '#ebe4d3',
  backgroundFocus: '#f0eadc',
  backgroundStrong: '#fbf6ec',

  color: '#1a1f2c',
  colorHover: '#0e1218',
  colorPress: '#3a4150',

  color1: '#f6f1e7',
  color2: '#f0eadc',
  color3: '#e8e1d0',
  color4: '#ddd5c1',
  color5: '#d0c8b3',
  color6: '#b8b09b',
  color7: '#9e9684',
  color8: '#7c746a',
  color9: '#5a6470',
  color10: '#3a4150',
  color11: '#252a35',
  color12: '#1a1f2c',

  accentBackground: '#b8742a',
  accentColor: '#f6f1e7',

  borderColor: '#ddd5c1',
  borderColorHover: '#d0c8b3',
  borderColorPress: '#ddd5c1',
  borderColorFocus: '#b8742a',

  shadowColor: 'rgba(26, 31, 44, 0.12)',
  shadowColorHover: 'rgba(26, 31, 44, 0.18)',
}

const generatedThemes = createThemes({
  base: {
    palette: {
      dark: darkPalette,
      light: lightPalette,
    },

    extra: {
      light: {
        ...Colors.blue,
        ...Colors.green,
        ...Colors.red,
        ...Colors.yellow,
        ...Colors.orange,
        ...Colors.purple,
        ...Colors.pink,
        ...Colors.gray,
        ...lightShadows,
        ...extraColors,
        ...lightSemanticColors,
        ...defaultStyleTokens,
        shadowColor: lightShadows.shadow1,
      },
      dark: {
        ...Colors.blueDark,
        ...Colors.greenDark,
        ...Colors.redDark,
        ...Colors.yellowDark,
        ...Colors.orangeDark,
        ...Colors.purpleDark,
        ...Colors.pinkDark,
        ...Colors.grayDark,
        ...darkShadows,
        ...extraColors,
        ...darkSemanticColors,
        ...defaultStyleTokens,
        shadowColor: darkShadows.shadow1,
      },
    },
  },

  accent: {
    palette: {
      dark: lightPalette,
      light: darkPalette,
    },
  },

  childrenThemes: {
    blue: {
      palette: {
        dark: Object.values(Colors.blueDark),
        light: Object.values(Colors.blue),
      },
    },
    red: {
      palette: {
        dark: Object.values(Colors.redDark),
        light: Object.values(Colors.red),
      },
    },
    yellow: {
      palette: {
        dark: Object.values(Colors.yellowDark),
        light: Object.values(Colors.yellow),
      },
    },
    green: {
      palette: {
        dark: Object.values(Colors.greenDark),
        light: Object.values(Colors.green),
      },
    },

    torch: {
      palette: {
        dark: torchDarkPalette,
        light: torchLightPalette,
      },
      extra: {
        light: {
          ...Colors.purple,
          ...lightShadows,
          ...extraColors,
          ...torchSemanticColorsLight,
          shadowColor: 'rgba(139, 92, 246, 0.1)',
        },
        dark: {
          ...Colors.purpleDark,
          ...darkShadows,
          ...extraColors,
          ...torchSemanticColorsDark,
          shadowColor: 'rgba(0, 0, 0, 0.5)',
        },
      },
    },

  },
})

// Spread base + theme overrides for Statio and Night Watch (matches the
// Pugio-Website pattern). createThemes' extra-token propagation is unreliable
// across child themes, so we merge flat overrides ourselves.
const allThemes = {
  ...generatedThemes,

  dark_statio: {
    ...generatedThemes.dark,
    ...statioDark,
  },
  light_statio: {
    ...generatedThemes.light,
    background: '#f4f2ec',
    backgroundHover: '#eae6dc',
    backgroundPress: '#ddd8cc',
    backgroundFocus: '#eae6dc',
    backgroundStrong: '#f8f6f0',
    color: '#0c0a06',
    color1: '#f4f2ec', color2: '#eae6dc', color3: '#ddd8cc', color4: '#d0cabc',
    color5: '#c4bcac', color6: '#a8a090', color7: '#8c8474', color8: '#706858',
    color9: '#544c3c', color10: '#3c3428', color11: '#241c14', color12: '#0c0a06',
    accentBackground: '#b08d3e',
    accentColor: '#f4f2ec',
    borderColor: '#d0cabc',
    borderColorHover: '#c4bcac',
    borderColorFocus: '#b08d3e',
  },

  dark_nightwatch: {
    ...generatedThemes.dark,
    ...nightWatchDark,
    ...nightWatchStyleTokens,
  },
  light_nightwatch: {
    ...generatedThemes.light,
    ...nightWatchLight,
    ...nightWatchStyleTokens,
  },
}

export type TamaguiThemes = typeof allThemes

export const themes: TamaguiThemes =
  process.env.TAMAGUI_ENVIRONMENT === 'client' &&
    process.env.NODE_ENV === 'production'
    ? ({} as any)
    : allThemes

// =============================================================================
// EXPORTED CONSTANTS
// =============================================================================

export const LABEL_COLOR_PRESETS = Object.values(labelColors)

export const LABEL_COLOR_MAP = labelColors
