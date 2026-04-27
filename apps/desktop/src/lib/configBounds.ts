// Validation bounds for AppConfig numeric fields.
//
// The sidecar will accept and persist any number, but values outside these
// bounds either break the probe loop (interval=0 freezes things), waste disk
// (retention=10000 days), or misrepresent the user's intent (timeout=5ms
// will fail every probe). We enforce bounds in the UI with inline errors so
// users see the constraint *before* they save.

export interface FieldBounds {
  min: number;
  max: number;
  /** If true, only whole numbers are valid. */
  integer: boolean;
  /** Short noun for error messages — "Ping interval must be at least…". */
  noun: string;
  /** Unit suffix for error messages — "seconds", "ms", "days". */
  unit: string;
}

export const CONFIG_BOUNDS = {
  ping_interval_sec: {
    min: 0.5,
    max: 300,
    integer: false,
    noun: 'Ping interval',
    unit: 'seconds',
  },
  ping_timeout_ms: {
    min: 100,
    max: 30000,
    integer: true,
    noun: 'Probe timeout',
    unit: 'ms',
  },
  flush_interval_sec: {
    min: 5,
    max: 3600,
    integer: true,
    noun: 'Flush interval',
    unit: 'seconds',
  },
  retention_raw_days: {
    min: 1,
    max: 365,
    integer: true,
    noun: 'Raw retention',
    unit: 'days',
  },
  retention_5min_days: {
    min: 1,
    max: 3650,
    integer: true,
    noun: '5-minute retention',
    unit: 'days',
  },
} as const satisfies Record<string, FieldBounds>;

export type ConfigNumericKey = keyof typeof CONFIG_BOUNDS;

export interface FieldValidation {
  /** The parsed value, only set when there's no error. */
  value?: number;
  /** Inline error message — undefined means valid. */
  error?: string;
}

/**
 * Validate a raw input string against a numeric field's bounds.
 *
 * Rules:
 *  - empty → "Required" (we don't allow blanking a config field)
 *  - non-numeric → "Must be a number"
 *  - non-integer when bounds say integer → "Whole numbers only"
 *  - below min → "At least <min> <unit>"
 *  - above max → "At most <max> <unit>"
 */
export function validateField(key: ConfigNumericKey, raw: string): FieldValidation {
  const bounds = CONFIG_BOUNDS[key];
  const trimmed = raw.trim();

  if (trimmed === '') {
    return { error: 'Required.' };
  }

  const n = bounds.integer ? Number.parseInt(trimmed, 10) : Number.parseFloat(trimmed);

  if (!Number.isFinite(n)) {
    return { error: 'Must be a number.' };
  }

  // parseInt('1.5') returns 1 silently — guard against that case.
  if (bounds.integer && !/^-?\d+$/.test(trimmed)) {
    return { error: 'Whole numbers only.' };
  }

  if (n < bounds.min) {
    return { error: `At least ${bounds.min} ${bounds.unit}.` };
  }

  if (n > bounds.max) {
    return { error: `At most ${bounds.max} ${bounds.unit}.` };
  }

  return { value: n };
}
