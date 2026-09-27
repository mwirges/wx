#!/usr/bin/env bash
# create-dmg.sh — Build a clean macOS DMG installer for wx.app
set -euo pipefail

APP_PATH="${1:-build/Build/Products/Release/wx.app}"
OUTPUT_DMG="${2:-../build/wx.dmg}"
VOL_NAME="${3:-wx}"

# Resolve paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MACOS_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

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

echo "==> Creating compressed disk image: ${OUTPUT_DMG}..."
if diskutil image create --help >/dev/null 2>&1; then
  diskutil image create from "${STAGE_DIR}" "${OUTPUT_DMG}" --format UDZO --volumeName "${VOL_NAME}"
else
  hdiutil create -volname "${VOL_NAME}" -srcfolder "${STAGE_DIR}" -ov -format UDZO "${OUTPUT_DMG}"
fi

echo "==> Successfully created DMG installer at: ${OUTPUT_DMG}"
ls -lh "${OUTPUT_DMG}"
