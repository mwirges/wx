#!/usr/bin/env bash
# create-dmg.sh — Build, sign, and package a macOS DMG installer for wx.app
set -euo pipefail

APP_PATH="${1:-build/Build/Products/Release/wx.app}"
OUTPUT_DMG="${2:-../build/wx.dmg}"
VOL_NAME="${3:-wx}"

# Resolve paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MACOS_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
ENTITLEMENTS="${MACOS_DIR}/WxMac/WxMac.entitlements"

# If relative path, check current dir first, then macos dir
if [[ "$APP_PATH" != /* ]]; then
  if [[ -d "$PWD/$APP_PATH" ]]; then
    APP_PATH="$PWD/$APP_PATH"
  else
    APP_PATH="${MACOS_DIR}/${APP_PATH}"
  fi
fi

if [[ "$OUTPUT_DMG" != /* ]]; then
  if [[ "$OUTPUT_DMG" == ../* ]] || [[ "$PWD" == "$MACOS_DIR" ]]; then
    OUTPUT_DMG="${MACOS_DIR}/${OUTPUT_DMG}"
  else
    OUTPUT_DMG="$PWD/$OUTPUT_DMG"
  fi
fi

if [[ ! -d "$APP_PATH" ]]; then
  echo "error: Application bundle not found at: $APP_PATH" >&2
  exit 1
fi

# 1. Determine code signing identity
if [[ -z "${CODE_SIGN_IDENTITY:-}" ]]; then
  # Auto-detect Developer ID Application certificate from Keychain
  CODE_SIGN_IDENTITY="$(security find-identity -v -p codesigning 2>/dev/null | grep "Developer ID Application:" | head -n 1 | sed -E 's/.*"([^"]+)".*/\1/' || true)"
fi

if [[ -n "${CODE_SIGN_IDENTITY}" ]]; then
  echo "==> Using Developer ID signing identity: ${CODE_SIGN_IDENTITY}"
  SIGN_FLAGS=(--force --options runtime --timestamp --sign "${CODE_SIGN_IDENTITY}")
else
  echo "==> Note: No Developer ID Application certificate found; signing ad-hoc (-)."
  echo "    For Gatekeeper distribution, a Developer ID certificate is required."
  SIGN_FLAGS=(--force --sign -)
fi

mkdir -p "$(dirname "$OUTPUT_DMG")"
rm -f "$OUTPUT_DMG"

STAGE_DIR="$(mktemp -d /tmp/wx-dmg-staging.XXXXXX)"
trap 'rm -rf "${STAGE_DIR}"' EXIT

echo "==> Staging wx.app into DMG payload..."
cp -R "$APP_PATH" "${STAGE_DIR}/wx.app"
ln -s /Applications "${STAGE_DIR}/Applications"

# Set volume icon if available
ICON_PATH="${MACOS_DIR}/WxMac/AppIcon.icns"
if [[ -f "$ICON_PATH" ]]; then
  cp "$ICON_PATH" "${STAGE_DIR}/.VolumeIcon.icns"
  if command -v SetFile >/dev/null 2>&1; then
    SetFile -c icnC "${STAGE_DIR}/.VolumeIcon.icns" 2>/dev/null || true
    SetFile -a C "${STAGE_DIR}" 2>/dev/null || true
  fi
fi

# 2. Sign embedded binaries, dylibs, and the app bundle
echo "==> Signing app binaries and payload..."
if [[ -f "${STAGE_DIR}/wx.app/Contents/MacOS/wx-cli" ]]; then
  codesign "${SIGN_FLAGS[@]}" "${STAGE_DIR}/wx.app/Contents/MacOS/wx-cli"
fi

# Sign any frameworks or dylibs if present
find "${STAGE_DIR}/wx.app/Contents" \( -name "*.framework" -o -name "*.dylib" \) 2>/dev/null | while read -r item; do
  codesign "${SIGN_FLAGS[@]}" "$item"
done

# Sign the main executable with entitlements
if [[ -f "$ENTITLEMENTS" ]]; then
  codesign "${SIGN_FLAGS[@]}" --entitlements "$ENTITLEMENTS" "${STAGE_DIR}/wx.app/Contents/MacOS/wx"
  codesign "${SIGN_FLAGS[@]}" --deep --entitlements "$ENTITLEMENTS" "${STAGE_DIR}/wx.app"
else
  codesign "${SIGN_FLAGS[@]}" "${STAGE_DIR}/wx.app"
fi

echo "==> Verifying wx.app code signature..."
codesign --verify --deep --strict --verbose=2 "${STAGE_DIR}/wx.app"

# 3. Create compressed disk image
echo "==> Creating compressed disk image: ${OUTPUT_DMG}..."
if diskutil image create --help >/dev/null 2>&1; then
  diskutil image create from "${STAGE_DIR}" "${OUTPUT_DMG}" --format UDZO --volumeName "${VOL_NAME}"
else
  hdiutil create -volname "${VOL_NAME}" -srcfolder "${STAGE_DIR}" -ov -format UDZO "${OUTPUT_DMG}"
fi

# 4. Set custom wx icon on the DMG file itself
if [[ -f "$ICON_PATH" ]]; then
  echo "==> Setting custom icon on ${OUTPUT_DMG}..."
  swift -e '
    import AppKit
    let args = CommandLine.arguments
    guard args.count >= 3 else { exit(1) }
    let iconPath = args[1]
    let targetPath = args[2]
    guard let img = NSImage(contentsOfFile: iconPath) else { exit(1) }
    _ = NSWorkspace.shared.setIcon(img, forFile: targetPath, options: [])
  ' "$ICON_PATH" "$OUTPUT_DMG" 2>/dev/null || true
fi

# 5. Sign the DMG itself
echo "==> Signing DMG disk image..."
if [[ -n "${CODE_SIGN_IDENTITY}" ]]; then
  codesign --force --sign "${CODE_SIGN_IDENTITY}" --timestamp "${OUTPUT_DMG}"
  echo "==> Verifying DMG code signature..."
  codesign --verify --verbose=2 "${OUTPUT_DMG}"
else
  codesign --force --sign - "${OUTPUT_DMG}" 2>/dev/null || true
fi

echo "==> Successfully created signed DMG installer at: ${OUTPUT_DMG}"
ls -lh "${OUTPUT_DMG}"

# 5. Optional Notarization
if [[ "${NOTARIZE:-0}" == "1" || "${NOTARIZE:-false}" == "true" ]]; then
  "${SCRIPT_DIR}/notarize-dmg.sh" "${OUTPUT_DMG}"
fi
