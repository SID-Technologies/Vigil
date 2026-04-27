import { useEffect, useState } from 'react';

interface PulsingDotProps {
  /** Base dot color (CSS color or var). */
  color: string;
  /** Diameter in px. */
  size?: number;
  /**
   * When this number changes, a new pulse ring is launched. Pass the cycle
   * counter from useProbeCycle / useLiveSamples so the dot pulses on every
   * probe cycle.
   */
  pulseKey?: number;
}

/**
 * Solid dot with an expanding ring pulse on each `pulseKey` change. Used
 * as the live "sidecar is alive" heartbeat in the StatusCard.
 *
 * The pulse ring is a separate element keyed by `pulseKey` so React unmounts
 * the previous ring and mounts a new one — that's what restarts the CSS
 * animation. Any in-flight pulse on the previous key is torn down so the
 * dot doesn't accumulate concurrent rings.
 */
export function PulsingDot({ color, size = 10, pulseKey = 0 }: PulsingDotProps) {
  // We keep one ring at a time. When pulseKey changes, force-remount via key.
  // The dot itself never re-renders unnecessarily.
  return (
    <span
      style={{
        position: 'relative',
        display: 'inline-block',
        width: size,
        height: size,
        verticalAlign: 'middle',
      }}
    >
      <span
        style={{
          position: 'absolute',
          inset: 0,
          borderRadius: '50%',
          backgroundColor: color,
          boxShadow: `0 0 6px ${color}`,
        }}
      />
      <PulseRing key={pulseKey} color={color} size={size} />
    </span>
  );
}

function PulseRing({ color, size }: { color: string; size: number }) {
  // Keyed remount → keyframe restarts. Animation is one-shot.
  // After 1.5s the ring is invisible (opacity 0); we leave it in the DOM
  // until the next pulse replaces it.
  const [mounted, setMounted] = useState(true);
  useEffect(() => {
    const id = setTimeout(() => setMounted(false), 1600);
    return () => clearTimeout(id);
  }, []);
  if (!mounted) return null;
  return (
    <span
      style={{
        position: 'absolute',
        inset: 0,
        width: size,
        height: size,
        borderRadius: '50%',
        backgroundColor: color,
        opacity: 0.7,
        animation: 'vigil-pulse-ring 1.5s ease-out forwards',
      }}
    />
  );
}
