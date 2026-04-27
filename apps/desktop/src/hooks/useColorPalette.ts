import { useMemo } from 'react';

/**
 * 13-color palette tuned for both Night Watch dark slate and the warm
 * parchment light theme. Avoids the brand amber (`var(--accentColor)`)
 * which is reserved for the median-line / accent / "current value" role —
 * if a target line shared that color, users would conflate "this target"
 * with "the live cursor."
 *
 * Colors picked to be:
 *   - Distinct in hue (every 30° around the wheel covered)
 *   - Mid-saturation so they don't compete with the amber accent
 *   - Readable on both #0b1116 and #f6f1e7 backgrounds
 *
 * Derived from Tableau 10 + Observable category10, hand-tuned.
 */
const PALETTE = [
  '#5b9bd5', // steel blue
  '#4caf50', // grass green
  '#ec407a', // pink
  '#26a69a', // teal
  '#ab47bc', // orchid
  '#ff7043', // coral
  '#5c6bc0', // indigo
  '#66bb6a', // lime green
  '#9575cd', // lavender
  '#00bcd4', // cyan
  '#d4af37', // muted gold (distinct from brand amber)
  '#78909c', // slate gray
  '#b71c1c', // deep red — reserved for highest-hash targets, gives error vibe
] as const;

/**
 * Deterministic FNV-1a-style hash. Same label always maps to same index,
 * so `router_icmp` is the same blue across reloads, page navigations, and
 * even between the dashboard and (future) history pages.
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
