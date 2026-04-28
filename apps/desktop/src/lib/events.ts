// Sidecar event listener helpers.
//
// The Tauri Rust shell forwards any line from the sidecar's stdout that has
// the shape `{"event": "...", "data": ...}` to a Tauri event with the same
// name. Frontend subscribes via @tauri-apps/api `listen()` and unsubscribes
// when components unmount.

import { listen, type UnlistenFn } from '@tauri-apps/api/event';

/**
 * Subscribe to a sidecar event. Returns an unlisten promise — await it once at
 * cleanup time. Typical pattern in a React component:
 *
 *   useEffect(() => {
 *     const unlistenPromise = onSidecarEvent('probe.cycle', (data) => { ... });
 *     return () => { unlistenPromise.then(fn => fn()); };
 *   }, []);
 */
export function onSidecarEvent<T>(
  event: string,
  handler: (data: T) => void,
): Promise<UnlistenFn> {
  return listen<T>(event, (msg) => handler(msg.payload));
}
