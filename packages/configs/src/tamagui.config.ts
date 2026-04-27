import { createTamagui, TamaguiConfig } from 'tamagui'
import { themes } from './themes'
import { spaceGroteskFont } from './fonts'
import { defaultConfig } from '@tamagui/config/v4'

export const config: TamaguiConfig = createTamagui({
  ...defaultConfig,
  themes,

  fonts: {
    ...defaultConfig.fonts, // Optionally keep default fonts like '$mono' if needed

    // Set 'Space Grotesk' as the primary body font
    body: spaceGroteskFont,

    // Set 'Space Grotesk' as the primary heading font (or define a separate one)
    heading: spaceGroteskFont,
  },
})

export default config
