# Vigil — Contributor Notes

A desktop network reliability monitor that watches when your ISP and
property manager won't. The name comes from the Roman *Vigiles* — Rome's
night watch. Built and maintained by SID Technologies, OSS under the
license at the repo root.

This file is the orientation doc for new contributors (human or AI).
Architecture, conventions, decisions, and the release runbook. For
end-user install / what-it-does, see `README.md`.

## Origin

Built because the maintainer's ISP and managed property could not give
straight answers about network reliability. Vigil generates the evidence
— uptime %, latency percentiles, outage timestamps, Wi-Fi signal
correlation — needed to confront them with facts.

The job: produce credible reliability evidence and make the proof
effortless to generate, dignified to share. The audience is technical-
leaning but not all developers — frustrated remote workers, friends with
bad home Wi-Fi, family members on the wrong side of a managed-property
contract.

## Architecture

**Tauri 2.x shell + Go sidecar.** Go does ~85% of the work; Rust is a
thin shell handling tray, window lifecycle, updater, and stdio IPC
bridge.

- **Sidecar** (`cmd/vigil-sidecar/`) — long-running Go process. Owns the
  probe loop, SQLite database (Ent ORM), aggregation, outage detection,
  retention pruning. Communicates with Tauri via newline-delimited JSON
  on stdin/stdout. Logs to a file in the OS app-data dir, *never* to
  stdout (stdout is reserved for IPC).
- **Tauri shell** (`apps/desktop/src-tauri/`) — spawns the sidecar at app
  start, bridges JSON IPC to frontend events. Tray menu, hide-on-close,
  auto-updater.
- **Frontend** (`apps/desktop/src/`) — React 19 + Tamagui. Tamagui config
  + theme controller live in `packages/configs/`.

## Conventions

- Go module: `github.com/sid-technologies/vigil`
- Logger: `rs/zerolog`
- Errors: structured wrapper in `pkg/errors/` (drop-in for stdlib
  `errors`, adds slog attrs)
- Buildinfo: `pkg/buildinfo/` reads `runtime/debug.BuildInfo()` for git
  commit/timestamp
- Storage: Ent ORM with auto-migration via `client.Schema.Create()`,
  pure-Go SQLite (`modernc.org/sqlite`, no cgo)
- IPC: stdio JSON-line, not loopback HTTP — avoids Windows Defender /
  macOS firewall prompts on first run
- Frontend: pnpm workspace, Tamagui `2.0.0-rc.36`, React `19.2.4`
- `packages/configs/` exports `@repo/configs/*` — Tamagui config, themes,
  theme controller, fonts

## Theme: Night Watch

Custom theme defined in `packages/configs/src/themes.ts`. Watchman's
tower at 2am: cold dark slate `#0b1116`, one warm watchfire amber
`#e0a458` burning. Vigil defaults to `nightwatch` style + dark mode. The
other styles (default / torch / odyssey) ship as alternatives in the
sidebar's theme picker.

## Aggregation tiers

Probe samples land in raw form, then roll up to coarser buckets on a
timer. Each tier has its own retention window:

| Tier         | Cadence  | Retention | Where built                |
|--------------|----------|-----------|----------------------------|
| raw          | 2.5 s    | 7 d       | direct from probe loop     |
| 1-min bucket | 1 min    | 14 d      | aggregator from raw        |
| 5-min bucket | 5 min    | 90 d      | aggregator from raw        |
| 1-h bucket   | 1 h      | forever   | aggregator from 5-min      |

Outages are detected live (3+ consecutive failures of one target or
every probe) and stored separately in their own table. Never aggregated
or pruned — the historical record is the whole point.

The 1-min tier exists because the 1h–6h chart band needs more detail
than 5-min (12–72 points) but less than raw (8–35k points). 60–360
points is the legibility sweet spot.

## Build & cross-compile

The sidecar must be cross-compiled per target before `tauri build`:

- `scripts/build-sidecar.sh` — host-platform sidecar build, drops into
  `apps/desktop/src-tauri/binaries/` with Tauri's platform-tagged naming
  convention.
- For release builds, GitHub Actions matrix builds on `macos-latest`
  (lipo'd universal) and `windows-latest`. Linux added later.

See `Makefile` for everyday commands (`make desktop-dev`,
`make desktop-build`, `make desktop-icons`).

## Key design decisions

1. **Tauri (Rust shell) + Go sidecar.** Rust learning curve avoided by
   writing only ~150 lines of config-shaped Rust — all real logic in Go.
2. **Auto-updater is required.** `tauri-plugin-updater` was the deciding
   factor over pure-Wails.
3. **stdio JSON-line IPC, not loopback HTTP.** Avoids first-run firewall
   prompts on Windows and macOS.
4. **Cross-platform from day one.** macOS + Windows + Linux. Linux Wi-Fi
   via `github.com/mdlayher/wifi` (pure Go netlink, no shell-out).
5. **Aggregation, not retention-only.** Raw 2.5s for 7 days, then
   coarser buckets. Keeps the SQLite database small over months of
   running while preserving long-term trends.
6. **Charts: recharts.** Time-scaled X axis with explicit null-fill at
   missing buckets so gaps in monitoring read as visible breaks instead
   of straight lines.
7. **13 default targets** (Google/Cloudflare DNS, Teams/Zoom/Outlook
   over ICMP+TCP, public STUN servers) seeded into `targets` on first
   run. Users can add/remove via the Targets page.

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
3. Apple ID → app-specific password → generate one labeled
   "vigil-notary".
4. Find your team ID at https://developer.apple.com/account → Membership.
5. Add to GitHub Secrets:
   - `APPLE_CERTIFICATE` = `base64 -i developer-id.p12 | pbcopy`
   - `APPLE_CERTIFICATE_PASSWORD` = the .p12 password
   - `APPLE_SIGNING_IDENTITY` = e.g. `Developer ID Application: <Name> (TEAMID)`
   - `APPLE_ID` = the Apple ID email
   - `APPLE_PASSWORD` = the app-specific password from step 3
   - `APPLE_TEAM_ID` = the 10-character team ID

### 3. Windows signing — pick one path

**Path A: Azure Trusted Signing** (recommended, ~$10/mo, no hardware token)

1. Sign up at https://learn.microsoft.com/en-us/azure/trusted-signing/.
2. Provision a code signing account + certificate profile.
3. Switch the GitHub Actions release workflow's signing step to the
   official Azure Trusted Signing action — `Azure/trusted-signing-action`.
4. Add Azure auth secrets per Azure docs.

**Path B: Standard OV cert** (~$100–200/yr from CertCentral, SSL.com,
etc.)

1. Buy an OV code-signing cert. Some vendors require an audio business
   verification call.
2. Download the .pfx with a strong password.
3. Add to GitHub Secrets:
   - `WINDOWS_CERTIFICATE` = `base64 -i cert.pfx | pbcopy`
   - `WINDOWS_CERTIFICATE_PASSWORD` = the .pfx password

   Caveat: SmartScreen will warn until reputation builds (~3000
   downloads). Fine for trusted users; friction for anonymous ones.

**Path C: skip Windows signing**

Leave the Windows secrets unset. The unsigned `.msi` will trigger a
Windows Defender "unrecognized publisher" warning that users will need
to click through. Acceptable for a beta among trusted users.

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

- macOS: download `Vigil_<version>_universal.dmg`, drag to Applications.
- Windows: download `Vigil_<version>_x64-setup.msi`, run installer.

Subsequent releases auto-update via `tauri-plugin-updater` — bump the
version in `apps/desktop/src-tauri/tauri.conf.json` and `package.json`,
tag, push.
