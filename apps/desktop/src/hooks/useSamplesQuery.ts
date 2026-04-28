import { useQuery } from '@tanstack/react-query';
import { samplesQuery, type Granularity, type SamplesQueryResult } from '../lib/ipc';

interface UseSamplesQueryParams {
  fromMs: number;
  toMs: number;
  granularity?: Granularity;
  targetLabels?: string[];
  /** Refetch interval in ms; pass 0 to disable. Defaults to 30s for live dashboards. */
  refetchIntervalMs?: number;
}

/**
 * Time-windowed sample query.
 *
 * Returns a discriminated union — branch on `data?.granularity` to know
 * whether you're rendering raw rows or aggregated buckets.
 */
export function useSamplesQuery(params: UseSamplesQueryParams) {
  const { fromMs, toMs, granularity = 'auto', targetLabels, refetchIntervalMs = 30_000 } = params;

  return useQuery<SamplesQueryResult>({
    queryKey: [
      'samples-query',
      fromMs,
      toMs,
      granularity,
      targetLabels?.slice().sort().join(',') ?? '',
    ],
    queryFn: () =>
      samplesQuery({
        from_ms: fromMs,
        to_ms: toMs,
        granularity,
        target_labels: targetLabels,
      }),
    refetchInterval: refetchIntervalMs > 0 ? refetchIntervalMs : false,
    // Keep prior data while a new fetch is in flight so live dashboards
    // don't flicker between refetches.
    placeholderData: (prev) => prev,
  });
}
