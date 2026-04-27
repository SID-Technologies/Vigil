import { useEffect } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { outagesList, type Outage } from '../lib/ipc';
import { onSidecarEvent } from '../lib/events';

interface UseOutagesParams {
  fromMs: number;
  toMs: number;
  scope?: string;
  onlyOpen?: boolean;
}

const KEY_PREFIX = ['outages-list'] as const;

/**
 * Outages list with auto-refetch on outage:start / outage:end events.
 * Combines polling (30s baseline, in case events are missed) with push
 * (instant on threshold crossing) for the freshest possible view.
 */
export function useOutages(params: UseOutagesParams) {
  const { fromMs, toMs, scope, onlyOpen } = params;
  const qc = useQueryClient();
  const queryKey = [...KEY_PREFIX, fromMs, toMs, scope ?? '', onlyOpen ?? false];

  const query = useQuery<Outage[]>({
    queryKey,
    queryFn: () =>
      outagesList({
        from_ms: fromMs,
        to_ms: toMs,
        scope,
        only_open: onlyOpen,
      }),
    refetchInterval: 30_000,
    placeholderData: (prev) => prev,
  });

  useEffect(() => {
    const startPromise = onSidecarEvent<Outage>('outage:start', () => {
      qc.invalidateQueries({ queryKey: KEY_PREFIX });
    });
    const endPromise = onSidecarEvent<Outage>('outage:end', () => {
      qc.invalidateQueries({ queryKey: KEY_PREFIX });
    });
    return () => {
      startPromise.then((fn) => fn()).catch(() => {});
      endPromise.then((fn) => fn()).catch(() => {});
    };
  }, [qc]);

  return query;
}

export type { Outage };
