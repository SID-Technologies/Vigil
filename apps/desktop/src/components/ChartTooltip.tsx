import type { ReactNode } from 'react';
import type { TooltipProps } from 'recharts';

interface ChartTooltipProps extends TooltipProps<number, string> {
  /** Optional formatter for the timestamp shown at the top. */
  formatLabel?: (ms: number) => string;
  /** Suffix appended to each value (e.g. "ms"). */
  unit?: string;
  /** Optional caption shown beneath the time, italic, gray. */
  caption?: ReactNode;
}

/**
 * Custom recharts Tooltip content — designed to make scrubbing on
 * multi-series charts predictable.
 *
 * Why this exists:
 *   recharts' default tooltip uses the underlying chart engine's hit-test
 *   logic, which can flicker between "show all series" and "show closest
 *   only" depending on where the cursor is relative to each line's
 *   render path. With 5+ overlapping target lines that's annoying.
 *
 * What this does differently:
 *   - Always lists EVERY series in the payload (recharts populates this
 *     with all dataKeys the cursor crosses) — no flicker.
 *   - Sorts rows by value descending so the worst (highest RTT) is at
 *     the top, which is what users care about when scrubbing.
 *   - Skips entries with nullish values cleanly (a target may have no
 *     data in that bucket).
 *   - Uses Night Watch palette colors directly, no fight with Tamagui
 *     class generation inside recharts' DOM subtree.
 */
export function ChartTooltip({ active, payload, label, formatLabel, unit = 'ms', caption }: ChartTooltipProps) {
  if (!active || !payload || payload.length === 0) return null;

  const labelText =
    typeof label === 'number' && formatLabel ? formatLabel(label) : String(label ?? '');

  // Recharts' payload entry: { value, name, color, dataKey, ... }.
  // Filter null/undefined values, sort desc by value.
  const rows = payload
    .filter((p) => p.value != null && Number.isFinite(p.value as number))
    .sort((a, b) => Number(b.value) - Number(a.value));

  if (rows.length === 0) return null;

  return (
    <div
      style={{
        background: 'var(--color2)',
        border: '1px solid var(--borderColor)',
        borderRadius: 6,
        padding: '8px 10px',
        fontSize: 11,
        minWidth: 180,
        // Drop shadow lifts the tooltip off the chart so it's readable
        // over dense line clusters.
        boxShadow: '0 4px 12px rgba(0, 0, 0, 0.4)',
        // Disable pointer events so cursor scrubbing isn't interrupted
        // when the tooltip overlaps the chart area.
        pointerEvents: 'none',
      }}
    >
      <div style={{ fontWeight: 600, color: 'var(--color12)', marginBottom: 4 }}>
        {labelText}
      </div>
      {caption ? (
        <div style={{ fontStyle: 'italic', color: 'var(--color8)', marginBottom: 6, fontSize: 10 }}>
          {caption}
        </div>
      ) : null}
      <div style={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
        {rows.map((r) => (
          <div
            key={String(r.dataKey)}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 6,
              fontVariantNumeric: 'tabular-nums',
            }}
          >
            <span
              style={{
                width: 8,
                height: 8,
                borderRadius: '50%',
                background: r.color || 'var(--accentColor)',
                flexShrink: 0,
              }}
            />
            <span style={{ flex: 1, color: 'var(--color11)' }}>{String(r.name ?? r.dataKey)}</span>
            <span
              style={{
                color: 'var(--color12)',
                fontWeight: 600,
                fontVariantNumeric: 'tabular-nums',
              }}
            >
              {Number(r.value).toFixed(2)} {unit}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}
