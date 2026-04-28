#!/usr/bin/env bash
#
# One-time setup: generate the Ed25519 keypair Tauri's updater uses to sign
# release manifests. Run this ONCE per project; the public key gets baked
# into tauri.conf.json, the private key goes into a GitHub Secret.
#
# Usage:
#
#   bash scripts/generate-updater-keys.sh
#
# Output:
#
#   ~/.tauri/vigil-updater-key             (private key — KEEP SECRET)
#   ~/.tauri/vigil-updater-key.pub         (public key — paste into tauri.conf.json)
#
# After running, update tauri.conf.json's plugins.updater.pubkey to the
# contents of the .pub file, and add these two secrets to GitHub:
#
#   TAURI_SIGNING_PRIVATE_KEY            = base64-encoded contents of ~/.tauri/vigil-updater-key
#   TAURI_SIGNING_PRIVATE_KEY_PASSWORD   = whatever passphrase you set below

set -euo pipefail

if ! command -v cargo >/dev/null 2>&1; then
  echo "Rust toolchain required. Install via https://rustup.rs/" >&2
  exit 1
fi

# tauri-cli ships the keygen subcommand. Install it locally if missing.
if ! cargo tauri --version >/dev/null 2>&1; then
  echo "==> Installing tauri-cli (one-time)..."
  cargo install tauri-cli@^2.0
fi

OUT_DIR="$HOME/.tauri"
mkdir -p "$OUT_DIR"

PRIV="$OUT_DIR/vigil-updater-key"
if [[ -f "$PRIV" ]]; then
  echo "Key already exists at $PRIV — refusing to overwrite."
  echo "Delete it manually if you really want to rotate."
  exit 1
fi

echo "==> Generating Ed25519 keypair"
echo "    You'll be prompted for a passphrase. Use a strong one — it's how you"
echo "    protect the private key. Save it in 1Password / your password manager."
cargo tauri signer generate -w "$PRIV"

echo ""
echo "==> Done."
echo ""
echo "Private key  : $PRIV"
echo "Public key   : $PRIV.pub"
echo ""
echo "Next steps:"
echo "  1. Copy the public-key string from $PRIV.pub into tauri.conf.json"
echo "     under plugins.updater.pubkey."
echo "  2. Add to GitHub repo Secrets:"
echo "       TAURI_SIGNING_PRIVATE_KEY            (base64-encode the private key file)"
echo "       TAURI_SIGNING_PRIVATE_KEY_PASSWORD   (the passphrase you just chose)"
echo ""
echo "     base64 -i $PRIV | pbcopy"
echo "     # paste into the GH secret value field"
echo ""
echo "  3. Commit the updated tauri.conf.json and push the v0.0.1 tag to"
echo "     trigger the first signed release."
