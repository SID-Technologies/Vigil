#!/usr/bin/env bash
#
# Build a macOS universal sidecar by lipo-ing the arm64 + amd64 builds.
#
# After this completes, Tauri's externalBin lookup finds:
#   binaries/vigil-sidecar-aarch64-apple-darwin    (arm64)
#   binaries/vigil-sidecar-x86_64-apple-darwin     (amd64)
#   binaries/vigil-sidecar-universal-apple-darwin  (lipo'd, optional)
#
# Tauri auto-picks per host arch. Shipping both individual targets is
# enough for native bundles; the universal binary is only useful if you
# want a single .app that runs on both architectures.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="$ROOT/apps/desktop/src-tauri/binaries"

if [[ "$(uname)" != "Darwin" ]]; then
  echo "lipo is macOS-only — run this on a Mac runner." >&2
  exit 1
fi

ARM="$OUT_DIR/vigil-sidecar-aarch64-apple-darwin"
INTEL="$OUT_DIR/vigil-sidecar-x86_64-apple-darwin"
UNIV="$OUT_DIR/vigil-sidecar-universal-apple-darwin"

if [[ ! -f "$ARM" || ! -f "$INTEL" ]]; then
  echo "==> Building both Mac architectures first"
  GOOS=darwin GOARCH=arm64 bash "$ROOT/scripts/build-sidecar.sh"
  GOOS=darwin GOARCH=amd64 bash "$ROOT/scripts/build-sidecar.sh"
fi

echo "==> lipo-ing universal binary"
lipo -create -output "$UNIV" "$ARM" "$INTEL"
echo "==> Built $(du -h "$UNIV" | cut -f1) → $UNIV"
file "$UNIV"
