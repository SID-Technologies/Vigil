import { useEffect, useState } from 'react';
import { check, type Update } from '@tauri-apps/plugin-updater';
import { relaunch } from '@tauri-apps/plugin-process';

interface UpdaterState {
  available: { version: string; currentVersion: string } | null;
  installing: boolean;
  /** Last error from a check or install attempt, surfaced for debug visibility. */
  error: string | null;
  install: () => Promise<void>;
  dismiss: () => void;
}

// Wait a beat after mount before pinging the update server — let the
// dashboard render first so a flaky network doesn't slow first paint.
const CHECK_DELAY_MS = 5_000;

export function useUpdater(): UpdaterState {
  const [update, setUpdate] = useState<Update | null>(null);
  const [installing, setInstalling] = useState(false);
  const [dismissed, setDismissed] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    const id = setTimeout(async () => {
      // Logged unconditionally so end users with the WebView devtools open
      // see whether the check ran. Cheap diagnostic for "the banner never
      // appeared" reports.
      console.log('[updater] checking for updates...');
      try {
        const found = await check();
        if (cancelled) return;
        if (found) {
          console.log(
            `[updater] update available: ${found.currentVersion} -> ${found.version}`,
          );
          setUpdate(found);
        } else {
          console.log('[updater] no update available');
        }
      } catch (err) {
        const message = err instanceof Error ? err.message : String(err);
        console.error('[updater] check failed:', message);
        if (!cancelled) setError(message);
      }
    }, CHECK_DELAY_MS);

    return () => {
      cancelled = true;
      clearTimeout(id);
    };
  }, []);

  const install = async () => {
    if (!update) return;
    setInstalling(true);
    setError(null);
    try {
      console.log('[updater] downloading + installing...');
      await update.downloadAndInstall();
      console.log('[updater] install complete, relaunching');
      await relaunch();
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      console.error('[updater] install failed:', message);
      setError(message);
      setInstalling(false);
    }
  };

  return {
    available:
      update && !dismissed
        ? { version: update.version, currentVersion: update.currentVersion }
        : null,
    installing,
    error,
    install,
    dismiss: () => setDismissed(true),
  };
}
