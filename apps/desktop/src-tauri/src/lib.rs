//! Vigil Tauri shell.
//!
//! Spawns the Go sidecar (`binaries/vigil-sidecar`) at app start, bridges
//! its stdio JSON-RPC protocol to the frontend via Tauri events, and exposes
//! a single `ipc_call` command for the frontend to make method calls into the
//! sidecar.
//!
//! Wire protocol (stdout from sidecar):
//! - `{"id": "...", "result": ...}`        → resolve pending request
//! - `{"id": "...", "error": {...}}`       → reject pending request
//! - `{"event": "name", "data": ...}`      → forward as Tauri event
//!
//! All real logic lives in the Go sidecar — this file is the bridge.

use std::collections::HashMap;
use std::sync::{Arc, Mutex};

use serde::{Deserialize, Serialize};
use tauri::{
    menu::{IsMenuItem, Menu, MenuItem, PredefinedMenuItem, Submenu},
    tray::TrayIconBuilder,
    Emitter, Manager, State,
};
use tauri_plugin_autostart::MacosLauncher;
use tauri_plugin_opener::OpenerExt;
use tauri_plugin_shell::process::{CommandChild, CommandEvent};
use tauri_plugin_shell::ShellExt;
use tokio::sync::oneshot;

// ============================================================================
// IPC types
// ============================================================================

#[derive(Debug, Serialize, Deserialize, Clone)]
struct IpcError {
    code: String,
    message: String,
}

/// Outgoing request written to sidecar stdin.
#[derive(Debug, Serialize)]
struct IpcRequest<'a> {
    id: &'a str,
    method: &'a str,
    #[serde(skip_serializing_if = "Option::is_none")]
    params: Option<&'a serde_json::Value>,
}

/// What we expect to read back. Either has `id` (a response) or `event` (an event).
#[derive(Debug, Deserialize)]
struct InboundRaw {
    id: Option<String>,
    result: Option<serde_json::Value>,
    error: Option<IpcError>,
    event: Option<String>,
    data: Option<serde_json::Value>,
}

type PendingMap = HashMap<String, oneshot::Sender<Result<serde_json::Value, IpcError>>>;

/// Shared state held by Tauri. The sidecar's stdin handle and the map of
/// in-flight request IDs both need to be reachable from the `ipc_call`
/// command and from the stdout reader task.
struct SidecarState {
    child: Mutex<Option<CommandChild>>,
    pending: Arc<Mutex<PendingMap>>,
}

impl SidecarState {
    fn new() -> Self {
        Self {
            child: Mutex::new(None),
            pending: Arc::new(Mutex::new(HashMap::new())),
        }
    }
}

// ============================================================================
// `ipc_call` command — the only Tauri command we expose. Frontend calls this
// with a method name + params; we forward to the sidecar and resolve with the
// response.
// ============================================================================

#[tauri::command]
async fn ipc_call(
    method: String,
    params: serde_json::Value,
    state: State<'_, SidecarState>,
) -> Result<serde_json::Value, IpcError> {
    let id = uuid::Uuid::new_v4().to_string();

    // Set up a oneshot to receive the response. Use std::sync::Mutex — locks
    // are never held across an .await, so blocking primitives are correct here
    // and sidestep tokio's "blocking_lock from within runtime" panic.
    let (tx, rx) = oneshot::channel();
    {
        let mut pending = state.pending.lock().expect("pending mutex poisoned");
        pending.insert(id.clone(), tx);
    }

    // Serialize and write the request to sidecar stdin.
    let req = IpcRequest {
        id: &id,
        method: &method,
        params: Some(&params),
    };
    let mut payload = match serde_json::to_vec(&req) {
        Ok(b) => b,
        Err(e) => {
            state.pending.lock().expect("pending mutex poisoned").remove(&id);
            return Err(IpcError {
                code: "marshal_failed".into(),
                message: e.to_string(),
            });
        }
    };
    payload.push(b'\n');

    // Acquire the child briefly to write. write() takes &mut self, so we
    // need an exclusive lock — writes are short, contention is fine.
    {
        let mut child_guard = state.child.lock().expect("child mutex poisoned");
        match child_guard.as_mut() {
            Some(child) => {
                if let Err(e) = child.write(&payload) {
                    drop(child_guard);
                    state.pending.lock().expect("pending mutex poisoned").remove(&id);
                    return Err(IpcError {
                        code: "sidecar_write_failed".into(),
                        message: e.to_string(),
                    });
                }
            }
            None => {
                drop(child_guard);
                state.pending.lock().expect("pending mutex poisoned").remove(&id);
                return Err(IpcError {
                    code: "sidecar_unavailable".into(),
                    message: "sidecar process not running".into(),
                });
            }
        }
    }

    // Await the response.
    match rx.await {
        Ok(Ok(value)) => Ok(value),
        Ok(Err(err)) => Err(err),
        Err(_) => Err(IpcError {
            code: "sidecar_dropped".into(),
            message: "response channel closed before reply".into(),
        }),
    }
}

// ============================================================================
// Sidecar spawn + stdout reader
// ============================================================================

fn spawn_sidecar(app: &tauri::AppHandle) -> Result<(), Box<dyn std::error::Error>> {
    // Ensure the per-app data dir exists; that's where the sidecar writes its
    // SQLite DB and rolling log file.
    let data_dir = app.path().app_data_dir()?;
    std::fs::create_dir_all(&data_dir)?;
    let data_dir_str = data_dir.to_string_lossy().to_string();

    log::info!("spawning sidecar with data-dir={}", data_dir_str);

    let sidecar = app
        .shell()
        .sidecar("vigil-sidecar")?
        .args(["--data-dir", &data_dir_str]);

    let (mut rx, child) = sidecar.spawn()?;

    // Stash the child handle so `ipc_call` can write to its stdin.
    {
        let state = app.state::<SidecarState>();
        let mut guard = state.child.lock().expect("child mutex poisoned");
        *guard = Some(child);
    }

    let app_handle = app.clone();
    tauri::async_runtime::spawn(async move {
        while let Some(event) = rx.recv().await {
            match event {
                CommandEvent::Stdout(bytes) => {
                    handle_sidecar_line(&app_handle, &bytes).await;
                }
                CommandEvent::Stderr(bytes) => {
                    let line = String::from_utf8_lossy(&bytes);
                    log::warn!("sidecar stderr: {}", line.trim_end());
                }
                CommandEvent::Terminated(payload) => {
                    log::warn!("sidecar terminated: code={:?}", payload.code);
                    // Drain pending requests so awaiters don't hang forever.
                    let state = app_handle.state::<SidecarState>();
                    let drained: Vec<_> = {
                        let mut pending = state.pending.lock().expect("pending mutex poisoned");
                        pending.drain().collect()
                    };
                    for (_, tx) in drained {
                        let _ = tx.send(Err(IpcError {
                            code: "sidecar_terminated".into(),
                            message: "sidecar exited before responding".into(),
                        }));
                    }
                    let _ = app_handle.emit("sidecar:terminated", payload.code);
                    break;
                }
                _ => {}
            }
        }
    });

    Ok(())
}

async fn handle_sidecar_line(app: &tauri::AppHandle, bytes: &[u8]) {
    let line = match std::str::from_utf8(bytes) {
        Ok(s) => s.trim(),
        Err(e) => {
            log::warn!("sidecar emitted non-utf8 stdout: {}", e);
            return;
        }
    };
    if line.is_empty() {
        return;
    }

    let parsed: InboundRaw = match serde_json::from_str(line) {
        Ok(v) => v,
        Err(e) => {
            // Sidecar should only ever write JSON to stdout. If we see something
            // else it's likely a panic stack trace or rogue print — log it.
            log::warn!("sidecar emitted non-JSON line: {} | parse err: {}", line, e);
            return;
        }
    };

    // Event path: forward to frontend listeners.
    if let Some(event_name) = parsed.event {
        let payload = parsed.data.unwrap_or(serde_json::Value::Null);
        if let Err(e) = app.emit(&event_name, payload) {
            log::warn!("failed to emit event '{}': {}", event_name, e);
        }
        return;
    }

    // Response path: fulfill the pending oneshot for this id.
    let Some(id) = parsed.id else {
        log::warn!("sidecar message has neither 'id' nor 'event': {}", line);
        return;
    };

    let state = app.state::<SidecarState>();
    let tx = {
        let mut pending = state.pending.lock().expect("pending mutex poisoned");
        pending.remove(&id)
    };
    let Some(tx) = tx else {
        log::warn!("response for unknown id: {}", id);
        return;
    };

    if let Some(err) = parsed.error {
        let _ = tx.send(Err(err));
    } else {
        let value = parsed.result.unwrap_or(serde_json::Value::Null);
        let _ = tx.send(Ok(value));
    }
}

// ============================================================================
// App menu (macOS menu bar / Windows + Linux in-window top bar)
// ============================================================================

/// Build and install the application menu. macOS shows it in the system menu
/// bar; Windows/Linux render it as the top bar inside the window.
///
/// Menu structure:
///
///   [App: Vigil]   About / Settings (⌘,) / Hide (⌘H) / Hide Others (⌘⌥H)
///                  / Show All / Quit (⌘Q)
///
///   [File]         New Report (⌘N) / Close Window (⌘W)
///
///   [Edit]         Undo / Redo / Cut / Copy / Paste / Select All
///                  (predefined — required for inputs to handle Cmd+C
///                   on macOS)
///
///   [View]         Reload (⌘R) / Toggle DevTools (⌘⌥I in dev) / —
///                  Dashboard (⌘1) / History (⌘2) / Outages (⌘3)
///                  / Targets (⌘4) / Settings (⌘,)
///
///   [Window]       Minimize (⌘M) / Zoom
///
///   [Help]         Vigil on GitHub
///
/// Menu items emit a Tauri event named `menu:select` with the item id as
/// the payload. The frontend's `useMenuEvents` hook (apps/desktop/src/
/// hooks/useMenuEvents.ts) listens and dispatches to the right handler
/// (navigate, open report modal, etc).
fn build_app_menu(app: &tauri::AppHandle) -> tauri::Result<Menu<tauri::Wry>> {

    // App submenu — first item, takes the bundle name on macOS automatically.
    let app_about = PredefinedMenuItem::about(app, Some("About Vigil"), None)?;
    let app_settings = MenuItem::with_id(app, "menu:settings", "Settings…", true, Some("CmdOrCtrl+,"))?;
    let app_hide = PredefinedMenuItem::hide(app, None)?;
    let app_hide_others = PredefinedMenuItem::hide_others(app, None)?;
    let app_show_all = PredefinedMenuItem::show_all(app, None)?;
    let app_quit = PredefinedMenuItem::quit(app, Some("Quit Vigil"))?;
    let app_submenu = Submenu::with_id_and_items(
        app,
        "menu:app",
        "Vigil",
        true,
        &[
            &app_about,
            &PredefinedMenuItem::separator(app)?,
            &app_settings,
            &PredefinedMenuItem::separator(app)?,
            &app_hide,
            &app_hide_others,
            &app_show_all,
            &PredefinedMenuItem::separator(app)?,
            &app_quit,
        ],
    )?;

    // File menu.
    let file_new_report = MenuItem::with_id(
        app,
        "menu:new_report",
        "New Report…",
        true,
        Some("CmdOrCtrl+N"),
    )?;
    let file_close = PredefinedMenuItem::close_window(app, None)?;
    let file_submenu = Submenu::with_id_and_items(
        app,
        "menu:file",
        "File",
        true,
        &[&file_new_report, &PredefinedMenuItem::separator(app)?, &file_close],
    )?;

    // Edit menu — predefined items are essential for input copy/paste.
    let edit_submenu = Submenu::with_id_and_items(
        app,
        "menu:edit",
        "Edit",
        true,
        &[
            &PredefinedMenuItem::undo(app, None)?,
            &PredefinedMenuItem::redo(app, None)?,
            &PredefinedMenuItem::separator(app)?,
            &PredefinedMenuItem::cut(app, None)?,
            &PredefinedMenuItem::copy(app, None)?,
            &PredefinedMenuItem::paste(app, None)?,
            &PredefinedMenuItem::select_all(app, None)?,
        ],
    )?;

    // View menu.
    let view_reload = MenuItem::with_id(app, "menu:reload", "Reload", true, Some("CmdOrCtrl+R"))?;
    #[cfg(debug_assertions)]
    let view_devtools = MenuItem::with_id(
        app,
        "menu:devtools",
        "Toggle Developer Tools",
        true,
        Some("CmdOrCtrl+Alt+I"),
    )?;
    let view_dashboard = MenuItem::with_id(app, "menu:nav:/", "Dashboard", true, Some("CmdOrCtrl+1"))?;
    let view_history = MenuItem::with_id(app, "menu:nav:/history", "History", true, Some("CmdOrCtrl+2"))?;
    let view_outages = MenuItem::with_id(app, "menu:nav:/outages", "Outages", true, Some("CmdOrCtrl+3"))?;
    let view_targets = MenuItem::with_id(app, "menu:nav:/targets", "Targets", true, Some("CmdOrCtrl+4"))?;

    // Build the view menu items list with the devtools entry only in debug builds.
    let mut view_items: Vec<&dyn IsMenuItem<_>> = vec![&view_reload];
    #[cfg(debug_assertions)]
    view_items.push(&view_devtools);
    let view_separator = PredefinedMenuItem::separator(app)?;
    view_items.push(&view_separator);
    view_items.push(&view_dashboard);
    view_items.push(&view_history);
    view_items.push(&view_outages);
    view_items.push(&view_targets);

    let view_submenu = Submenu::with_id_and_items(app, "menu:view", "View", true, &view_items)?;

    // Window menu — predefined Minimize is the macOS convention.
    let window_submenu = Submenu::with_id_and_items(
        app,
        "menu:window",
        "Window",
        true,
        &[
            &PredefinedMenuItem::minimize(app, None)?,
            &PredefinedMenuItem::maximize(app, None)?,
            &PredefinedMenuItem::separator(app)?,
            &PredefinedMenuItem::fullscreen(app, None)?,
        ],
    )?;

    // Help menu.
    let help_github = MenuItem::with_id(app, "menu:help_github", "Vigil on GitHub", true, None::<&str>)?;
    let help_submenu = Submenu::with_id_and_items(
        app,
        "menu:help",
        "Help",
        true,
        &[&help_github],
    )?;

    Menu::with_items(
        app,
        &[
            &app_submenu,
            &file_submenu,
            &edit_submenu,
            &view_submenu,
            &window_submenu,
            &help_submenu,
        ],
    )
}

/// Handle menu events. Most items emit a `menu:select` Tauri event with
/// the item id; the frontend listens and dispatches. A few items handle
/// directly here because they're Rust-only (reload, devtools, github).
fn handle_menu_event(app: &tauri::AppHandle, id: &str) {
    match id {
        "menu:reload" => {
            if let Some(window) = app.get_webview_window("main") {
                if let Err(e) = window.eval("window.location.reload()") {
                    log::warn!("reload failed: {}", e);
                }
            }
        }
        "menu:devtools" => {
            // Devtools methods only exist on Tauri's WebviewWindow when the
            // `devtools` Cargo feature is enabled — Tauri auto-enables it in
            // debug builds but not in release. Gate accordingly so the menu
            // item is a no-op in shipped builds rather than a compile error.
            #[cfg(debug_assertions)]
            if let Some(window) = app.get_webview_window("main") {
                if window.is_devtools_open() {
                    window.close_devtools();
                } else {
                    window.open_devtools();
                }
            }
        }
        "menu:help_github" => {
            if let Err(e) = app.opener().open_url("https://github.com/dan-flanagan/Vigil", None::<&str>) {
                log::warn!("open github failed: {}", e);
            }
        }
        // Everything else (nav targets, settings, new report) is forwarded
        // to the frontend, which knows how to navigate / open modals.
        other => {
            if let Err(e) = app.emit("menu:select", other) {
                log::warn!("emit menu:select failed: {}", e);
            }
        }
    }
}

// ============================================================================
// Tray
// ============================================================================

fn build_tray(app: &tauri::AppHandle) -> Result<(), Box<dyn std::error::Error>> {
    let show = MenuItem::with_id(app, "show", "Show Vigil", true, None::<&str>)?;
    let hide = MenuItem::with_id(app, "hide", "Hide", true, None::<&str>)?;
    let open_data = MenuItem::with_id(app, "open_data", "Open data folder", true, None::<&str>)?;
    let toggle_autostart = MenuItem::with_id(
        app,
        "toggle_autostart",
        "Launch on login",
        true,
        None::<&str>,
    )?;
    let separator = PredefinedMenuItem::separator(app)?;
    let quit = MenuItem::with_id(app, "quit", "Quit Vigil", true, None::<&str>)?;
    let menu = Menu::with_items(
        app,
        &[
            &show,
            &hide,
            &separator,
            &open_data,
            &toggle_autostart,
            &separator,
            &quit,
        ],
    )?;

    // macOS menu-bar template image. Flat black silhouette with alpha;
    // macOS inverts to white in dark menu bars and tints to accent color
    // when the menu is open. icon_as_template(true) flips ".icon()" into
    // "treat as monochrome, tint per system theme." On Windows / Linux
    // the same PNG is used directly (no auto-tint there).
    let tray_icon = tauri::include_image!("./icons/tray-template.png");

    TrayIconBuilder::new()
        .menu(&menu)
        // On macOS the click event is consumed by menu display by default,
        // BUT only when this is explicitly true. Some Tauri 2.x versions
        // default to false (left-click does literally nothing if you also
        // attach an on_tray_icon_event handler). Set explicitly.
        .show_menu_on_left_click(true)
        .icon(tray_icon)
        .icon_as_template(true)
        .tooltip("Vigil — Network Watch")
        .on_menu_event(|app, event| {
            match event.id.as_ref() {
            "show" => {
                if let Some(window) = app.get_webview_window("main") {
                    let _ = window.show();
                    let _ = window.set_focus();
                }
            }
            "hide" => {
                if let Some(window) = app.get_webview_window("main") {
                    let _ = window.hide();
                }
            }
            "open_data" => {
                // Open the data folder via the opener plugin (replaced
                // tauri-plugin-shell's deprecated open()).
                let app = app.clone();
                tauri::async_runtime::spawn(async move {
                    if let Ok(dir) = app.path().app_data_dir() {
                        let path_str = dir.to_string_lossy().to_string();
                        if let Err(e) = app.opener().open_path(&path_str, None::<&str>) {
                            log::warn!("failed to open data folder: {}", e);
                        }
                    }
                });
            }
            "toggle_autostart" => {
                // Toggle the autostart plugin's enabled state. The first time
                // this runs it'll prompt for permission on macOS.
                use tauri_plugin_autostart::ManagerExt;
                let manager = app.autolaunch();
                let enabled = manager.is_enabled().unwrap_or(false);
                let res = if enabled { manager.disable() } else { manager.enable() };
                if let Err(e) = res {
                    log::warn!("autostart toggle failed: {}", e);
                }
            }
            "quit" => {
                app.exit(0);
            }
            _ => {}
            }
        })
        // Note: deliberately no on_tray_icon_event handler. macOS convention
        // for tray-resident utilities (NordPass, 1Password mini, Wi-Fi,
        // Bluetooth) is "click → show menu, period." The "Show Vigil" item
        // in the menu is how users open the window. Adding a click→show
        // handler here would compete with show_menu_on_left_click(true) and
        // produce jarring double-action behavior.
        .build(app)?;

    Ok(())
}

// ============================================================================
// Entry point
// ============================================================================

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_opener::init())
        .plugin(tauri_plugin_updater::Builder::new().build())
        .plugin(tauri_plugin_process::init())
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_autostart::init(
            // AppleScript launcher on macOS — most reliable across Sonoma+.
            // On Windows/Linux this argument is ignored.
            MacosLauncher::AppleScript,
            // Args to pass to vigil at autostart. None — we just launch normally.
            Some(vec![]),
        ))
        .plugin(tauri_plugin_single_instance::init(|app, _argv, _cwd| {
            // Bring the existing window forward when a second launch is attempted.
            if let Some(window) = app.get_webview_window("main") {
                let _ = window.show();
                let _ = window.set_focus();
            }
        }))
        .manage(SidecarState::new())
        // Menu event handler stays on the builder — it's a global handler
        // that fires for any installed menu's items.
        .on_menu_event(|app, event| {
            handle_menu_event(app, event.id().0.as_str());
        })
        .setup(|app| {
            let handle = app.handle();

            // Build + install the app menu explicitly. The .menu(closure)
            // builder method is unreliable in some Tauri 2.x versions on
            // macOS — explicit set_menu in setup is the canonical pattern.
            match build_app_menu(handle) {
                Ok(menu) => {
                    if let Err(e) = handle.set_menu(menu) {
                        log::warn!("set_menu failed: {}", e);
                    }
                }
                Err(e) => log::warn!("build_app_menu failed: {}", e),
            }

            build_tray(handle)?;
            if let Err(e) = spawn_sidecar(handle) {
                log::error!("failed to spawn sidecar: {}", e);
                // Keep the app open — `ipc_call` will surface a clear error
                // to the frontend instead of crashing.
            }
            Ok(())
        })
        .on_window_event(|window, event| {
            // Hide to tray on close instead of quitting.
            if let tauri::WindowEvent::CloseRequested { api, .. } = event {
                let _ = window.hide();
                api.prevent_close();
            }
        })
        .invoke_handler(tauri::generate_handler![ipc_call])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
