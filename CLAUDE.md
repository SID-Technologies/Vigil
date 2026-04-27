# Vigil — Project Context

Vigil is a SID Technologies project: a desktop network reliability monitor that watches when your ISP and property manager won't. The name comes from the Roman *Vigiles* — Rome's night watch.

This is **not** on the official SID product roadmap (Torch, Statio, Denarius, JanusLedger). It's a personal tool that happens to follow SID conventions for consistency.

## Origin

Built because the founder's ISP and managed property could not give straight answers about network reliability. The tool generates the evidence — uptime %, latency percentiles, outage timestamps, Wi-Fi signal correlation — needed to confront them with facts.

Started life as a Python CLI (`src/pingscraper/`). Mid-rewrite to a Go-backed Tauri desktop app to:
1. Make it installable for non-technical people (the original target users — friends, neighbors with the same ISP problems).
2. Run continuously in the tray, not in a terminal window.
3. Aggregate samples (5-min and 1-hour rollups) so the SQLite database stays small over months of running.
4. Let users add/remove probe targets via UI without editing code.

## Architecture

**Tauri 2.x shell + Go sidecar.** Go does ~85% of the work; Rust is a thin shell handling tray, window lifecycle, updater, and stdio IPC bridge.

- **Sidecar** (`cmd/vigil-sidecar/`) — long-running Go process. Owns the probe loop, SQLite database (Ent ORM), aggregation, outage detection, retention pruning. Communicates with Tauri via newline-delimited JSON on stdin/stdout. Logs to a file in the OS app-data dir, *never* to stdout (stdout is reserved for IPC).
- **Tauri shell** (`apps/desktop/src-tauri/`) — spawns the sidecar at app start, bridges JSON IPC to frontend events. Tray menu, hide-on-close, auto-updater.
- **Frontend** (`apps/desktop/src/`) — React 19 + Tamagui. Tamagui config + theme controller live in `packages/configs/` (mirrored from Pugio).

## Conventions inherited from Pugio (Statio)

This project deliberately mirrors Pugio's structure so SID patterns stay consistent:

- Go module: `github.com/sid-technologies/vigil`
- Logger: `rs/zerolog`
- Errors: structured wrapper in `pkg/errors/` (drop-in for stdlib `errors`, adds slog attrs)
- Buildinfo: `pkg/buildinfo/` reads `runtime/debug.BuildInfo()` for git commit/timestamp
- Ent ORM with auto-migration via `client.Schema.Create()` (added in phase 2)
- pnpm workspace, Tamagui `2.0.0-rc.36`, React `19.2.4`, all pinned exactly to Pugio versions
- `packages/configs/` exports `@repo/configs/*` — Tamagui config, themes, theme controller, fonts

Pugio things deliberately omitted:
- No platform-core integration (Vigil is a local desktop tool, no auth/billing/orgs)
- No OpenAPI / ogen (stdio IPC, no HTTP API surface)
- No Cobra/viper/TOML config (sidecar takes one CLI arg `--data-dir`; settings live in the DB)
- No multi-service `services/*` layout (single sidecar in `cmd/vigil-sidecar/`)
- No Postgres/pgx (SQLite via `modernc.org/sqlite` — pure Go, no cgo)

## Theme: Night Watch

Custom theme defined in `packages/configs/src/themes.ts`. Watchman's tower at 2am: cold dark slate, one warm amber light burning. Vigil defaults to `nightwatch` style + dark mode. The other Pugio styles (default/torch/retro/odyssey) remain available via the toggle.

## Build & cross-compile

The sidecar must be cross-compiled per target before `tauri build`:

- `scripts/build-sidecar.sh` — host-platform sidecar build, drops into `apps/desktop/src-tauri/binaries/` with Tauri's platform-tagged naming convention.
- For release builds, GitHub Actions matrix builds on `macos-latest` (lipo'd universal) and `windows-latest`. Linux added later.

## Conversation history (decisions made before code was written)

1. **Framework choice:** Tauri (Rust shell) + Go sidecar. Rust learning curve avoided by writing only ~150 lines of config-shaped Rust — all real logic in Go.
2. **Auto-updater required.** Tauri's `tauri-plugin-updater` chosen specifically for this; this is why we picked Tauri over pure-Wails.
3. **IPC: stdio JSON-line, not loopback HTTP.** Avoids Windows Defender / macOS firewall prompts on first run.
4. **Cross-platform from day 1.** macOS + Windows + Linux. Linux Wi-Fi via `github.com/mdlayher/wifi` (pure Go netlink, no shell-out).
5. **Aggregation tiers:** raw (7d retention) → 5-min (90d) → 1-hour (forever). Outages detected live and stored separately, never re-aggregated.
6. **Charts:** recharts. The Python tool's HTML reports use Chart.js — we'll replace at port time.
7. **Default targets:** same 13 as the Python `DEFAULT_TARGETS` (Google/Cloudflare DNS, Teams/Zoom/Outlook over ICMP+TCP, public STUN servers). Seeded into `targets` table on first run.

## Phase plan

- **Phase 1 (current):** Skeleton + IPC pipeline. Empty Go sidecar with `health.check`, Tauri shell that spawns it and shows "Connected — version" in the window. Validates the entire pipeline before any real logic.
- **Phase 2:** Probe engine + storage. All four probe types ported, Ent schemas, monitor + flusher goroutines, default targets seeding.
- **Phase 3:** Aggregation + outages + retention.
- **Phase 4:** Live dashboard with real-time charts.
- **Phase 5:** History, Outages, Targets, Settings pages.
- **Phase 6:** Reports (CSV/JSON/HTML), tray polish, empty/error states.
- **Phase 7:** Cross-compile pipeline, signing, notarization, GH Actions release, auto-updater wiring.

Each phase ends with a runnable artifact. Don't skip phases.

## Release runbook

The CI machinery is in place (`.github/workflows/{ci,release}.yml`,
`scripts/{build-sidecar,lipo-darwin,generate-updater-keys}.sh`). What
remains is one-time human setup that requires real-world accounts.

### 1. Generate updater keys (one-time, ~5 min)

```bash
bash scripts/generate-updater-keys.sh
```

Follow the printed instructions. Paste the public key into
`apps/desktop/src-tauri/tauri.conf.json` under `plugins.updater.pubkey`.
Add `TAURI_SIGNING_PRIVATE_KEY` and `TAURI_SIGNING_PRIVATE_KEY_PASSWORD`
to GitHub Secrets.

### 2. macOS signing + notarization (one-time, ~1–2 hours of waiting)

1. Enroll in the Apple Developer Program ($99/yr) at
   https://developer.apple.com/programs/. Approval can take a day.
2. In the Apple Developer portal: Certificates → create a "Developer ID
   Application" certificate. Download as `.p12` with a strong password.
3. Apple ID → app-specific password → generate one labeled "vigil-notary".
4. Find your team ID at https://developer.apple.com/account → Membership.
5. Add to GitHub Secrets:
   - `APPLE_CERTIFICATE` = `base64 -i developer-id.p12 | pbcopy` (paste output)
   - `APPLE_CERTIFICATE_PASSWORD` = the .p12 password
   - `APPLE_SIGNING_IDENTITY` = e.g. `Developer ID Application: Dan Flanagan (TEAMID)`
   - `APPLE_ID` = your Apple ID email
   - `APPLE_PASSWORD` = the app-specific password from step 3
   - `APPLE_TEAM_ID` = the 10-character team ID

### 3. Windows signing — pick one path

**Path A: Azure Trusted Signing** (recommended, ~$10/mo, no hardware token)

1. Sign up at https://learn.microsoft.com/en-us/azure/trusted-signing/.
2. Provision a code signing account + certificate profile.
3. Switch the GitHub Actions release workflow's signing step to the
   official Azure Trusted Signing action — `Azure/trusted-signing-action`.
4. Add Azure auth secrets per Azure docs.

**Path B: Standard OV cert** (~$100–200/yr from CertCentral, SSL.com, etc.)

1. Buy an OV code-signing cert. Some vendors require an audio business
   verification call.
2. Download the .pfx with a strong password.
3. Add to GitHub Secrets:
   - `WINDOWS_CERTIFICATE` = `base64 -i cert.pfx | pbcopy`
   - `WINDOWS_CERTIFICATE_PASSWORD` = the .pfx password

   Caveat: SmartScreen will warn until reputation builds (~3000 downloads).
   For users you trust, fine. For anonymous strangers, friction.

**Path C: skip Windows signing** (smallest scope)

Leave the Windows secrets unset. The unsigned `.msi` will trigger a
Windows Defender "unrecognized publisher" warning that your friends and
family will need to click through. Acceptable for a beta among trusted
users.

### 4. Generate icons (one-time)

Drop a 1024×1024 PNG at `apps/desktop/app-icon.png`, then:

```bash
make desktop-icons
```

This produces all required sizes + .icns + .ico. Commit the icons.

### 5. First release

```bash
git tag v0.0.1
git push origin v0.0.1
```

The release workflow fires, builds on macos-latest + windows-latest,
signs/notarizes, uploads bundles to a GitHub Release draft. Edit the
release body, then click Publish.

End users install:
- macOS: download `Vigil_0.0.1_universal.dmg`, drag to Applications.
- Windows: download `Vigil_0.0.1_x64-setup.msi`, run installer.

Subsequent releases auto-update via tauri-plugin-updater — bump the
version in `apps/desktop/src-tauri/tauri.conf.json` and `package.json`,
tag, push.
