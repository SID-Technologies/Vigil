#!/usr/bin/env bash
#
# Build the Vigil Go sidecar for the host platform and drop it into Tauri's
# externalBin directory with the platform-tagged name Tauri 2.x expects.
#
# Tauri's `externalBin` config in tauri.conf.json points at:
#   apps/desktop/src-tauri/binaries/vigil-sidecar
#
# At build time, Tauri looks for a per-target file matching:
#   vigil-sidecar-<rustc-target-triple>[<.exe>]
#
# So this script writes BOTH the bare name (used by `tauri dev`) AND the
# triple-suffixed name (used by `tauri build`).
#
# For cross-compiling to other platforms in CI, pass GOOS/GOARCH explicitly:
#   GOOS=darwin  GOARCH=amd64 bash scripts/build-sidecar.sh
#   GOOS=darwin  GOARCH=arm64 bash scripts/build-sidecar.sh
#   GOOS=windows GOARCH=amd64 bash scripts/build-sidecar.sh

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="$ROOT/apps/desktop/src-tauri/binaries"
mkdir -p "$OUT_DIR"

# Resolve the target triple from GOOS/GOARCH (default = host).
GOOS="${GOOS:-$(go env GOOS)}"
GOARCH="${GOARCH:-$(go env GOARCH)}"

case "$GOOS-$GOARCH" in
  darwin-amd64)  TRIPLE="x86_64-apple-darwin"        ; EXT="" ;;
  darwin-arm64)  TRIPLE="aarch64-apple-darwin"       ; EXT="" ;;
  linux-amd64)   TRIPLE="x86_64-unknown-linux-gnu"   ; EXT="" ;;
  linux-arm64)   TRIPLE="aarch64-unknown-linux-gnu"  ; EXT="" ;;
  windows-amd64) TRIPLE="x86_64-pc-windows-msvc"     ; EXT=".exe" ;;
  windows-arm64) TRIPLE="aarch64-pc-windows-msvc"    ; EXT=".exe" ;;
  *)
    echo "unsupported GOOS/GOARCH: $GOOS/$GOARCH" >&2
    exit 1
    ;;
esac

OUT_TRIPLE="$OUT_DIR/vigil-sidecar-$TRIPLE$EXT"

echo "==> Building Vigil sidecar"
echo "    GOOS=$GOOS GOARCH=$GOARCH"
echo "    target=$TRIPLE"
echo "    out=$OUT_TRIPLE"

cd "$ROOT"

# Stripped, statically-relocatable binary. -trimpath strips machine-local paths
# from the binary so reproducible builds work.
GOOS="$GOOS" GOARCH="$GOARCH" \
  go build \
    -trimpath \
    -ldflags="-s -w -X github.com/sid-technologies/vigil/pkg/buildinfo.version=${VIGIL_VERSION:-dev}" \
    -o "$OUT_TRIPLE" \
    ./cmd/vigil-sidecar

echo "==> Built $(du -h "$OUT_TRIPLE" | cut -f1) → $OUT_TRIPLE"
