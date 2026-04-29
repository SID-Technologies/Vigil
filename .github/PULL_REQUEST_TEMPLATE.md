<!--
Thanks for sending a PR. A few things that make review faster:

1. Read .github/CONTRIBUTING.md if you haven't.
2. One PR per concern. Refactor + feature in the same diff is hard to review.
3. CI must be green. `make lint` and `pnpm tsc --noEmit` locally save a round trip.
-->

## What

<!-- One or two sentences describing the change. -->

## Why

<!-- The problem it solves, the user it helps, the bug it fixes. Link the issue if there is one (`Closes #123`). -->

## How to verify

<!--
Concrete steps a reviewer can run:
  - `make build-sidecar && pnpm tauri dev`
  - Click X, expect Y.
  - Tests: `go test ./internal/...` / `pnpm test`
-->

## Screenshots / recordings

<!-- For UI changes, before/after screenshots. For dashboard or chart changes, a short screen recording. -->

## Checklist

- [ ] Tests added or updated where it matters
- [ ] `make lint` passes (Go) and `pnpm tsc --noEmit` passes (frontend)
- [ ] No new comments that just describe what the code does
- [ ] If a user-facing string changed, screenshots above show the new text
