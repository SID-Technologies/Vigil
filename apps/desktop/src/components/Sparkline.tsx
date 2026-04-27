interface SparklineProps {
  values: number[];
  color: string;
  /** Width in CSS px. Match the parent's effective inner width. */
  width?: number;
  /** Height in CSS px. */
  height?: number;
  strokeWidth?: number;
  /** Optionally fill area under the line. */
  filled?: boolean;
}

/**
 * Inline SVG sparkline. Lightweight on purpose — recharts inside a tile
 * grid is overkill (each instance is a full React subtree). This is one
 * <svg> per tile, ~30 lines.
 *
 * Auto-scales Y to data min/max with a 5% pad so flat lines don't render
 * along the very top/bottom edge.
 *
 * Renders nothing if fewer than 2 points (a single point isn't a line).
 */
export function Sparkline({
  values,
  color,
  width = 80,
  height = 22,
  strokeWidth = 1.5,
  filled = false,
}: SparklineProps) {
  if (values.length < 2) {
    return <svg width={width} height={height} aria-hidden />;
  }

  const min = Math.min(...values);
  const max = Math.max(...values);
  const range = max - min || 1;
  const pad = range * 0.05;

  const points = values.map((v, i) => {
    const x = (i / (values.length - 1)) * width;
    const y = height - ((v - min + pad) / (range + 2 * pad)) * height;
    return `${x.toFixed(2)},${y.toFixed(2)}`;
  });

  return (
    <svg width={width} height={height} aria-hidden style={{ overflow: 'visible' }}>
      {filled && (
        <polygon
          points={`0,${height} ${points.join(' ')} ${width},${height}`}
          fill={color}
          opacity={0.15}
        />
      )}
      <polyline
        points={points.join(' ')}
        fill="none"
        stroke={color}
        strokeWidth={strokeWidth}
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}
