# Vigil

Continuous network reliability monitor with a desktop UI. Pings router, anycast DNS, and real call-quality endpoints (Teams, Zoom, Outlook) on a fixed interval. Samples Wi-Fi signal. Detects outages. Generates the receipts you hand your ISP or property manager when they tell you the network is fine.

Works on macOS, Windows, and Linux. Lives in the system tray.

## Status

This repo is mid-rewrite from the original Python CLI to a Go + Tauri desktop app.

- **Legacy Python tool** (`src/pingscraper/`, `tests/`, `pyproject.toml`) — original implementation, still runnable via `uv run pingscraper`. Reference for the port.
- **Go sidecar** (`cmd/`, `internal/`, `pkg/`, `db/`) — replacement engine. Embedded into the desktop app as a stdio JSON-RPC sidecar.
- **Desktop app** (`apps/desktop/`) — Tauri 2.x shell + React 19 + Tamagui frontend. Tray-resident, hides on close.
- **Shared TS configs** (`packages/configs/`) — Tamagui config, theme controller, fonts. Mirrored from Pugio.

The Python tool will be removed once the rewrite reaches feature parity.

## Quick start (development)

Prereqs:
- Go 1.25+
- Node 20+ and pnpm 10+
- Rust toolchain (rustup), Tauri 2.x prerequisites for your OS — see https://tauri.app/start/prerequisites/

```bash
# Install JS deps
pnpm install

# Build the Go sidecar for the host platform and drop it where Tauri expects it
make build-sidecar

# Run the desktop app in dev mode (Vite + Tauri)
make desktop-dev
```

## Repo layout

```
cmd/vigil-sidecar/        Entry point for the Go sidecar
internal/                 Sidecar internals (probes, monitor, IPC, storage)
pkg/                      Reusable Go packages (errors, log, buildinfo)
db/                       Ent schemas + generated code (added in phase 2)
apps/desktop/             Tauri shell + React frontend
packages/configs/         Tamagui config, theme controller, fonts
scripts/                  Build helpers
src/pingscraper/          Legacy Python (reference only — to be removed)
```

## Make targets

Run `make help` to see them all. The important ones:

- `make build-sidecar` — Cross-compile Go sidecar for the host platform.
- `make desktop-dev` — Start Vite + Tauri in dev mode (depends on `build-sidecar`).
- `make desktop-build` — Production bundle (.dmg / .msi / AppImage).
- `make lint` — Run all linters via pre-commit.
- `make test` — Run Go tests.
