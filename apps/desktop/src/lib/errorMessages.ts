// Translation layer between sidecar IPC error codes and user-facing copy.
//
// The sidecar emits stable machine-readable codes ("invalid_params",
// "not_found", etc) along with a developer-grade message. The UI
// shouldn't put either of those in front of users — they're either
// jargon ("invalid_params") or boilerplate from a third-party lib
// ("ent: constraint failed: UNIQUE...").
//
// VigilIpcError uses this map to give Error.message a friendly,
// watchman-voice translation. The original sidecar message is kept on
// `.technicalDetail` for debugging / log inspection.

export const ERROR_MESSAGES: Record<string, string> = {
  // ---- Generic ----
  invalid_params:
    "Something in that request didn't look right. Check the values and try again.",
  internal:
    "Something went sideways on our end. If this keeps happening, the sidecar log will have details.",
  not_found: "Couldn't find what you asked for — it may have been removed.",

  // ---- Reports ----
  window_too_large:
    "That's a lot of data — Vigil caps reports at 90 days. Try a shorter range.",

  // ---- Targets ----
  builtin_immutable:
    "Built-in targets are protected. You can disable them, but they can't be edited or deleted.",

  // ---- IPC bridge errors (Tauri shell ↔ sidecar) ----
  sidecar_unavailable:
    "Vigil's sidecar isn't running. Restart the app to bring it back.",
  sidecar_terminated:
    "Vigil's sidecar stopped unexpectedly. Restart the app to recover.",
  sidecar_dropped:
    "Lost contact with the sidecar mid-request. The next attempt should reconnect.",
  sidecar_write_failed:
    "Couldn't send that request to the sidecar — it may have just stopped.",
  marshal_failed: "Couldn't prepare that request. This shouldn't happen — please report it.",
  method_not_found:
    "Vigil doesn't recognize that command. The sidecar may be a different version than the app.",
};

/**
 * Translate a sidecar error to user-friendly copy. Falls back to the
 * sidecar's original message if we don't have a translation registered —
 * which is OK because new error codes show up as raw text and we can
 * add them to this map later.
 */
export function translateError(code: string, fallback: string): string {
  return ERROR_MESSAGES[code] ?? fallback;
}
