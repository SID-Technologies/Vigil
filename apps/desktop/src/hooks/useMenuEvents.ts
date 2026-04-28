import { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';

import { onSidecarEvent } from '../lib/events';

interface UseMenuEventsOptions {
  /** Called when File → New Report is selected, or Cmd+N pressed. */
  onNewReport?: () => void;
}

/**
 * Subscribes to the `menu:select` Tauri event emitted by the Rust app menu.
 * Dispatches frontend actions:
 *
 *   menu:nav:<path>     → navigate(path) via react-router
 *   menu:settings       → navigate('/settings') (Cmd+,)
 *   menu:new_report     → invoke the onNewReport callback (page-owned)
 *
 * Mount this hook once at the App-level so menu shortcuts work regardless
 * of which page is active. Use `onNewReport` to wire the report modal —
 * since the modal is owned by the History page, App.tsx forwards by
 * navigating to /history with a `?report=1` query param that the History
 * page detects on mount.
 */
export function useMenuEvents(options: UseMenuEventsOptions = {}) {
  const navigate = useNavigate();

  useEffect(() => {
    const unlisten = onSidecarEvent<string>('menu:select', (id) => {
      if (id.startsWith('menu:nav:')) {
        const path = id.slice('menu:nav:'.length);
        navigate(path);
        return;
      }
      if (id === 'menu:settings') {
        navigate('/settings');
        return;
      }
      if (id === 'menu:new_report') {
        if (options.onNewReport) {
          options.onNewReport();
          return;
        }
        // No handler registered — surface via History page query param.
        navigate('/history?report=1');
        return;
      }
    });
    return () => {
      unlisten.then((fn) => fn()).catch(() => {});
    };
  }, [navigate, options]);
}
