#!/usr/bin/env bash
# notarize-dmg.sh — Submit macOS DMG to Apple Notary Service and staple the ticket
set -euo pipefail

DMG_PATH="${1:-build/wx.dmg}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Resolve relative DMG path
if [[ "$DMG_PATH" != /* ]]; then
  if [[ -f "$PWD/$DMG_PATH" ]]; then
    DMG_PATH="$PWD/$DMG_PATH"
  elif [[ -f "${ROOT_DIR}/${DMG_PATH}" ]]; then
    DMG_PATH="${ROOT_DIR}/${DMG_PATH}"
  fi
fi

if [[ ! -f "$DMG_PATH" ]]; then
  echo "error: DMG file not found at: $DMG_PATH" >&2
  exit 1
fi

echo "==> Checking signature on ${DMG_PATH}..."
if ! codesign -dvvv "$DMG_PATH" 2>&1 | grep "Developer ID Application:" >/dev/null; then
  echo "error: ${DMG_PATH} is not signed with a Developer ID certificate." >&2
  echo "Run 'make mac-dmg' first to build and sign with Developer ID." >&2
  exit 1
fi

# Detect notarization credentials
NOTARY_ARGS=()
PROFILE="${NOTARY_PROFILE:-wx-notary}"

# 1. Check if explicit Keychain profile or default wx-notary profile exists
if xcrun notarytool history --keychain-profile "$PROFILE" >/dev/null 2>&1; then
  NOTARY_ARGS=(--keychain-profile "$PROFILE")
  echo "==> Using notarytool keychain profile: ${PROFILE}"
elif [[ -n "${NOTARY_APPLE_ID:-}" && -n "${NOTARY_PASSWORD:-}" ]]; then
  TEAM_ID="${NOTARY_TEAM_ID:-RB77BW6U32}"
  NOTARY_ARGS=(--apple-id "$NOTARY_APPLE_ID" --password "$NOTARY_PASSWORD" --team-id "$TEAM_ID")
  echo "==> Using Apple ID credentials for team ${TEAM_ID}..."
elif [[ -n "${NOTARY_KEY:-}" && -n "${NOTARY_KEY_ID:-}" && -n "${NOTARY_ISSUER:-}" ]]; then
  NOTARY_ARGS=(--key "$NOTARY_KEY" --key-id "$NOTARY_KEY_ID" --issuer "$NOTARY_ISSUER")
  echo "==> Using App Store Connect API Key ${NOTARY_KEY_ID}..."
else
  echo ""
  echo "=========================================================================="
  echo "  Apple Notarization Credentials Not Found"
  echo "=========================================================================="
  echo "To notarize wx.dmg, store your credentials in the macOS Keychain once:"
  echo ""
  echo "  xcrun notarytool store-credentials \"${PROFILE}\" \\"
  echo "    --apple-id <your-apple-id@email.com> \\"
  echo "    --team-id RB77BW6U32"
  echo ""
  echo "  (Generate an App-Specific Password at https://appleid.apple.com)"
  echo ""
  echo "Alternatively, you can provide environment variables:"
  echo "  export NOTARY_APPLE_ID=\"you@example.com\""
  echo "  export NOTARY_PASSWORD=\"xxxx-xxxx-xxxx-xxxx\""
  echo "  export NOTARY_TEAM_ID=\"RB77BW6U32\""
  echo ""
  echo "Once configured, re-run:"
  echo "  make mac-notarize"
  echo "or:"
  echo "  make mac-dmg NOTARIZE=1"
  echo "=========================================================================="
  echo ""
  exit 1
fi

echo "==> Submitting ${DMG_PATH} to Apple Notary Service (this usually takes 30-60s)..."
xcrun notarytool submit "$DMG_PATH" "${NOTARY_ARGS[@]}" --wait

echo "==> Stapling notarization ticket to ${DMG_PATH}..."
xcrun stapler staple "$DMG_PATH"

echo "==> Validating staple..."
xcrun stapler validate "$DMG_PATH"

echo "==> Verifying with Gatekeeper..."
spctl -a -vvv -t install "$DMG_PATH"

echo ""
echo "==> SUCCESS: ${DMG_PATH} is fully signed, notarized, stapled, and Gatekeeper-approved!"
