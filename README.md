# Vigil

Continuous network reliability monitor with a desktop UI. Pings router, anycast DNS, and real call-quality endpoints (Teams, Zoom, Outlook) on a fixed interval. Samples Wi-Fi signal. Detects outages. Generates the receipts you hand your ISP or property manager when they tell you the network is fine.

Works on macOS, Windows, and Linux. Lives in the system tray.

## Install

Download the latest release from [GitHub Releases](https://github.com/sid-technologies/vigil/releases/latest). Pick the file for your OS:

| OS | File | Notes |
|---|---|---|
| macOS (Apple Silicon + Intel) | `Vigil_<version>_universal.dmg` | Universal binary, runs on both architectures |
| Windows 10/11 (x64) | `Vigil_<version>_x64-setup.msi` | Installer |
| Linux (Debian / Ubuntu) | `vigil_<version>_amd64.deb` | `sudo dpkg -i vigil_*.deb` |
| Linux (everything else) | `vigil_<version>_amd64.AppImage` | `chmod +x` then run |

### macOS

1. Open the `.dmg` and drag **Vigil.app** to your **Applications** folder.
2. Open Applications, double-click Vigil.

**If you see "Vigil can't be opened because Apple cannot check it for malicious software":**
that means this build is unsigned. Open **System Settings → Privacy & Security**, scroll to the bottom, and click **"Open Anyway"** next to the Vigil notice. Confirm in the prompt that follows. After this first time, Vigil opens normally on every launch.

(Signed builds skip this step entirely. Releases tagged after we complete Apple Developer enrollment will be code-signed and notarized.)

### Windows

1. Run the `.msi` installer. Defender SmartScreen may show **"Windows protected your PC"** if the build is unsigned.
2. Click **More info → Run anyway**.
3. Walk through the installer (one screen, no choices to make).
4. Vigil launches automatically; check the system tray (bottom-right corner).

### Linux

**.deb (Ubuntu, Debian, Mint, Pop!_OS):**

```bash
sudo dpkg -i vigil_*_amd64.deb
sudo apt-get install -f   # if dependency errors, this resolves them
```

**.AppImage (Fedora, Arch, NixOS, anything else):**

```bash
chmod +x Vigil_*_amd64.AppImage
./Vigil_*_amd64.AppImage
```

ICMP probes use unprivileged ICMP sockets. On Ubuntu/Fedora this works without sudo for any user; on locked-down distros you may need to add yourself to `net.ipv4.ping_group_range`.

## First launch

After Vigil opens:

1. **The dashboard begins probing immediately** — every 2.5 seconds, 13 default targets (router + Google/Cloudflare DNS + Teams/Zoom/Outlook + STUN). Within 5 seconds you'll see "Last cycle: 13/13 ok" if everything's healthy.
2. **Vigil lives in the tray** — close the window and it keeps probing in the background. Right-click the tray icon for Show / Hide / Open data folder / Launch on login / Quit.
3. **First 5-minute aggregation appears at minute ~6** — the sidecar collects raw samples and rolls them into 5-min buckets. The dashboard chart populates as buckets close.
4. **Outages auto-detect** — three consecutive failures of the same target trigger an outage event. Lose Wi-Fi for 10 seconds and the dashboard goes red.
5. **Generate a report** any time — History page → "Generate report" button. Pick CSV / JSON / HTML, choose a folder. The HTML report is a self-contained dashboard you can email to your ISP.

Data lives in:

- **macOS**: `~/Library/Application Support/dev.vigil.desktop/`
- **Windows**: `%APPDATA%\dev.vigil.desktop\`
- **Linux**: `~/.local/share/dev.vigil.desktop/`

The tray menu has an **"Open data folder"** shortcut.

## Development

This repo is mid-rewrite from the original Python CLI to a Go + Tauri desktop app.

- **Legacy Python tool** (`src/pingscraper/`, `tests/`, `pyproject.toml`) — original implementation, still runnable via `uv run pingscraper`. Reference for the port.
- **Go sidecar** (`cmd/`, `internal/`, `pkg/`, `db/`) — replacement engine. Embedded into the desktop app as a stdio JSON-RPC sidecar.
- **Desktop app** (`apps/desktop/`) — Tauri 2.x shell + React 19 + Tamagui frontend. Tray-resident, hides on close.
- **Shared TS configs** (`packages/configs/`) — Tamagui config, theme controller, fonts. Mirrored from Pugio.

The Python tool will be removed once the rewrite reaches feature parity.

### Prereqs

- Go 1.25+
- Node 20+ and pnpm 10+
- Rust toolchain (rustup), Tauri 2.x prerequisites for your OS — see https://tauri.app/start/prerequisites/

### Quick start

```bash
# Install JS deps
pnpm install

# Generate Ent code (only needed first time / after schema changes)
make gen-ent

# Build the Go sidecar for the host platform and drop it where Tauri expects it
make build-sidecar

# Run the desktop app in dev mode (Vite + Tauri)
make desktop-dev
```

### Repo layout

```
cmd/vigil-sidecar/        Entry point for the Go sidecar
internal/                 Sidecar internals (probes, monitor, IPC, storage)
pkg/                      Reusable Go packages (errors, log, buildinfo)
db/                       Ent schemas + generated code
apps/desktop/             Tauri shell + React frontend
packages/configs/         Tamagui config, theme controller, fonts
scripts/                  Build helpers
src/pingscraper/          Legacy Python (reference only — to be removed)
```

### Make targets

Run `make help` to see them all. The important ones:

- `make gen-ent` — Run Ent codegen after editing schemas in `db/ent/schema/`.
- `make build-sidecar` — Cross-compile Go sidecar for the host platform.
- `make desktop-dev` — Start Vite + Tauri in dev mode (depends on `build-sidecar`).
- `make desktop-build` — Production bundle (.dmg / .msi / AppImage).
- `make desktop-icons` — Regenerate all Tauri icon sizes from `apps/desktop/app-icon.png`.
- `make lint` — Run all linters via pre-commit.
- `make test` — Run Go tests.

## License

MIT.
