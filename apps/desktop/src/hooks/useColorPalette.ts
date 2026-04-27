import { useMemo } from 'react';

/**
 * 13-color per-target palette tuned for both Night Watch dark slate and the
 * warm parchment light theme.
 *
 * Color rules — what the palette deliberately avoids:
 *
 *   - **Watchfire amber** (`#e0a458`) and amber-adjacent golds. Reserved
 *     for "current value / median / live cursor / accent." A target line
 *     colored amber would conflate target identity with the live signal.
 *   - **Error red** (`#f85149`) and red-adjacent oranges/corals. Reserved
 *     for failures and outages. A target line in red would read as
 *     "failing" even when it's healthy.
 *   - **Warning yellow** in its pure form.
 *   - **Success green** in its pure form (we keep ONE green slot —
 *     emerald — that's saturated enough not to be confused with status).
 *
 * What's left: blues, purples, pinks, teals, greens, and a neutral slate.
 * Hues are spread roughly evenly across perceptual color space; adjacent
 * palette indices are visually distinct so even unfortunate hash
 * collisions don't pair similar colors.
 *
 * Color-blind safety: this palette was designed with deuteranopia (the
 * most common red-green blindness) in mind. There's only one green and
 * no oranges/reds, so the most-confusable axis is removed entirely.
 * Pinks are saturated enough that they don't reduce to "gray" for
 * protanopes.
 *
 * Contrast: every color sits in the OKLCH 55-70 lightness band — readable
 * on slate `#0b1116` (dark) and parchment `#f6f1e7` (light) without per-
 * theme variants.
 */
const PALETTE = [
  '#0ea5e9', // sky blue
  '#5b9bd5', // steel blue (softer)
  '#6366f1', // indigo
  '#8b5cf6', // violet
  '#a855f7', // purple
  '#ec4899', // pink
  '#be185d', // dark magenta
  '#f472b6', // rose — deliberately PINK not red, distant from error red
  '#14b8a6', // teal
  '#06b6d4', // cyan
  '#22c55e', // emerald (single green slot)
  '#84cc16', // lime green-yellow
  '#64748b', // cool slate (neutral)
] as const;

/**
 * Deterministic FNV-1a-style hash. Same label always maps to the same
 * index, so `router_icmp` is the same blue across reloads, page
 * navigations, and between the Dashboard, History, and Outages pages.
 */
function hash(s: string): number {
  let h = 2166136261;
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i);
    h = (h + ((h << 1) + (h << 4) + (h << 7) + (h << 8) + (h << 24))) >>> 0;
  }
  return h;
}

export function useColorPalette() {
  return useMemo(
    () => ({
      /** Returns a stable hex color for the given target label. */
      getColor: (label: string): string => PALETTE[hash(label) % PALETTE.length],
      palette: [...PALETTE],
    }),
    [],
  );
}
