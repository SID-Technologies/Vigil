# Vigil app icons

Generate icons by running:

```bash
# From the repo root, drop a 1024×1024 PNG at apps/desktop/app-icon.png, then:
make desktop-icons
```

This invokes `cargo tauri icon` which produces all required sizes:

- `32x32.png`
- `128x128.png`
- `128x128@2x.png`
- `icon.icns` (macOS bundle icon)
- `icon.ico` (Windows installer icon)
- `icon.png` (used for the system tray)

These files must exist before `tauri build` will succeed. For development
(`tauri dev`) Tauri will fall back to a default icon if these are missing.
