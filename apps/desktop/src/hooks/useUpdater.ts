import { useEffect, useState } from 'react';
import { check, type Update } from '@tauri-apps/plugin-updater';
import { relaunch } from '@tauri-apps/plugin-process';

interface UpdaterState {
  available: { version: string; currentVersion: string } | null;
  installing: boolean;
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

  useEffect(() => {
    let cancelled = false;

    const id = setTimeout(async () => {
      try {
        const found = await check();
        if (!cancelled && found) setUpdate(found);
      } catch {
        // Most common cause: no internet. Silently no-op — the user finds
        // out about updates the next time they relaunch with connectivity.
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
    try {
      await update.downloadAndInstall();
      await relaunch();
    } catch (err) {
      console.error('updater: install failed', err);
      setInstalling(false);
    }
  };

  return {
    available:
      update && !dismissed
        ? { version: update.version, currentVersion: update.currentVersion }
        : null,
    installing,
    install,
    dismiss: () => setDismissed(true),
  };
}
