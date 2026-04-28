import { useCallback, useEffect, useState } from 'react';

const STORAGE_KEY = 'vigil-dashboard-selection';

function loadFromStorage(): string[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw);
    return Array.isArray(parsed) ? parsed.filter((x) => typeof x === 'string') : [];
  } catch {
    return [];
  }
}

function persist(labels: string[]) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(labels));
  } catch {
    // Quota / private mode — selection just won't survive reload. Not fatal.
  }
}

/**
 * Multi-select state for target tiles, persisted to localStorage so user
 * choices survive reload. State synchronizes across tabs of the same app
 * via the `storage` event — a nice-to-have if the user opens DevTools and
 * has the app showing twice.
 *
 * Empty selection means "no per-target lines — show median across all"
 * which is the dashboard default and the right behavior for "is the
 * network ok overall."
 */
export function useTargetSelection() {
  const [labels, setLabels] = useState<string[]>(loadFromStorage);

  // Cross-tab sync (rare but free).
  useEffect(() => {
    const onStorage = (e: StorageEvent) => {
      if (e.key !== STORAGE_KEY) return;
      setLabels(loadFromStorage());
    };
    window.addEventListener('storage', onStorage);
    return () => window.removeEventListener('storage', onStorage);
  }, []);

  const setBoth = useCallback((next: string[]) => {
    setLabels(next);
    persist(next);
  }, []);

  const toggle = useCallback(
    (label: string) => {
      setLabels((prev) => {
        const next = prev.includes(label)
          ? prev.filter((l) => l !== label)
          : [...prev, label];
        persist(next);
        return next;
      });
    },
    [],
  );

  const clear = useCallback(() => setBoth([]), [setBoth]);

  const setAll = useCallback((all: string[]) => setBoth(all), [setBoth]);

  const isSelected = useCallback(
    (label: string) => labels.includes(label),
    [labels],
  );

  return { labels, toggle, clear, setAll, isSelected };
}
