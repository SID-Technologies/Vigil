# Contributing to Vigil

Vigil is a small project. There's no formal review board, no SLA on PRs, and no community
manager to escalate to — it's one maintainer working in evenings. Keeping that in mind makes
contribution smoother for everyone.

## Before you open something

- **Bugs**: open an issue with the bug-report template. The sidecar log path is printed in
  the template — please attach it.
- **Features**: open an issue *before* writing code. Half the time the answer is "yes but
  smaller" or "no because <reason that isn't obvious from the README>", and finding that
  out before you've spent a weekend on it is good for both of us.
- **Questions / "is this a bug?"**: GitHub Discussions, not Issues.

## Layout

```
cmd/vigil-sidecar/        Go sidecar entrypoint
internal/                 sidecar internals (probes, ipc, monitor, storage)
db/ent/                   Ent ORM schemas + generated code
pkg/                      reusable Go packages (errors, buildinfo)
apps/desktop/             Tauri 2.x desktop app
  src/                    React 19 + Tamagui UI
  src-tauri/              Rust shell (tray, window, IPC bridge)
packages/configs/         Shared Tamagui config + Night Watch theme
scripts/                  Build / cross-compile helpers
```

The Go sidecar does ~85% of the work. The Rust shell is intentionally thin — tray, window,
auto-updater, stdio bridge. Most contributions land in `internal/` (Go) or `apps/desktop/src/`
(TS/React).

## Setup

You need:

- **Go** 1.24 or newer
- **Node** 20.x and **pnpm** 10.15.0 (`corepack enable`)
- **Rust** stable (only needed if you touch `apps/desktop/src-tauri/`)
- **pre-commit** (`pip install pre-commit` or `brew install pre-commit`)

Then:

```bash
make install            # pnpm workspace deps
make install-tools      # pre-commit hooks
make desktop-dev        # builds sidecar, launches Tauri dev
```

## Running it

| | |
|---|---|
| Sidecar only | `go run ./cmd/vigil-sidecar --data-dir /tmp/vigil-dev` |
| Desktop dev | `make desktop-dev` |
| Desktop release build | `make desktop-build` |
| Tests | `make test` |
| Lint | `make lint` (runs all pre-commit hooks) |
| Frontend typecheck | `cd apps/desktop && pnpm tsc --noEmit` |

After changing an Ent schema in `db/ent/schema/`, run `make gen-ent`.

## Code conventions

- **Go errors**: use `pkg/errors.Wrap` / `Wrapf`, not `fmt.Errorf`. The `forbidigo` linter
  enforces this. The wrapper is a drop-in replacement for stdlib `errors`.
- **Logger**: `rs/zerolog`. No fmt.Println in shipping code.
- **Comments**: write *why*, not *what*. Identifiers should explain what; comments are for
  hidden constraints, surprising invariants, and historical context. PR review will flag
  comments that just narrate the code.
- **Frontend**: Tamagui components, not raw HTML, for anything that needs the Night Watch
  theme. Use `var(--token)` only inside `style={}`, not in className strings.
- **No new dependencies without a reason in the PR description.** This applies to both Go
  and npm sides.

## Tests

- New Go code that has interesting branching gets table-driven tests.
- The Ent migrator isn't goroutine-safe; the test helper in `db/test_util.go` serializes
  `Schema.Create()` across `t.Parallel()` tests. Use it.
- Frontend tests aren't currently set up (April 2026). If you want to add Vitest, that's a
  welcome PR — please align with the Pugio / Torch conventions.

## Commit messages

Conventional-ish, but not strict: `feat:`, `fix:`, `refactor:`, `chore:`, `docs:`. Keep the
subject under ~70 chars; put the *why* in the body.

```
fix(probes): treat ICMP timeout as transient, not unreachable

The Linux kernel returns EAGAIN for raw ICMP packets when the route
table is being rebuilt (DHCP renewal). We were classifying that as a
hard "unreachable" and creating false outage records.
```

## Pull requests

- One concern per PR. Refactor + feature in the same diff is hard to review and slow to land.
- CI must be green. `make lint` + `pnpm tsc --noEmit` locally before pushing saves round trips.
- If the change is user-facing (UI, a flag, an error message), include a screenshot or a
  short recording. The PR template has a slot for this.
- The maintainer may rebase or squash before merging. That's normal.

## License

By contributing you agree that your contribution will be licensed under the [MIT License](../LICENSE)
that covers the project.
