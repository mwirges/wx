#!/usr/bin/env bash
# Check build/WxCore.xcframework: platforms, headers, module map, exports.
# Missing any required piece exits nonzero.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
FW="${ROOT}/build/WxCore.xcframework"
PLIST="${FW}/Info.plist"

fail() {
  echo "ios-framework-test: $*" >&2
  exit 1
}

[[ -f "$PLIST" ]] || fail "missing ${PLIST}"

plutil -lint "$PLIST" >/dev/null

# Newline-separated LibraryIdentifier values from one tool.
collect_ids() {
  local tool="$1"
  local i=0
  local id
  local out=""
  while true; do
    if [[ "$tool" == "buddy" ]]; then
      id="$(/usr/libexec/PlistBuddy -c "Print :AvailableLibraries:${i}:LibraryIdentifier" "$PLIST" 2>/dev/null)" || break
    else
      id="$(plutil -extract "AvailableLibraries.${i}.LibraryIdentifier" raw -o - "$PLIST" 2>/dev/null)" || break
    fi
    id="${id//$'\n'/}"
    out="${out}${id}"$'\n'
    i=$((i + 1))
  done
  printf '%s' "$out"
}

require_id() {
  local source="$1"
  local needle="$2"
  local ids="$3"
  if ! printf '%s\n' "$ids" | grep -qx "$needle"; then
    fail "${source}: Info.plist does not list ${needle}"
  fi
}

buddy_ids="$(collect_ids buddy)"
plutil_ids="$(collect_ids plutil)"
require_id PlistBuddy ios-arm64 "$buddy_ids"
require_id PlistBuddy ios-arm64-simulator "$buddy_ids"
require_id plutil ios-arm64 "$plutil_ids"
require_id plutil ios-arm64-simulator "$plutil_ids"

check_slice() {
  local id="$1"
  local slice="${FW}/${id}"
  local headers="${slice}/Headers"
  local map="${headers}/module.modulemap"
  local lib nm_out nlibs

  [[ -d "$slice" ]] || fail "${id}: slice directory missing"
  [[ -d "$headers" ]] || fail "${id}: missing Headers"
  [[ -f "$map" ]] || fail "${id}: missing Headers/module.modulemap"
  [[ -f "${headers}/libwxcore.h" ]] || fail "${id}: missing generated header Headers/libwxcore.h"
  if ! grep -q 'WxRunJSON' "${headers}/libwxcore.h" || ! grep -q 'WxFreeString' "${headers}/libwxcore.h"; then
    fail "${id}: generated header does not declare WxRunJSON and WxFreeString"
  fi
  if ! grep -q 'WxSetCacheDir' "${headers}/libwxcore.h"; then
    fail "${id}: generated header does not declare WxSetCacheDir"
  fi
  if ! grep -q '^module WxCore {$' "$map" || ! grep -q 'header "libwxcore.h"' "$map"; then
    fail "${id}: module.modulemap must declare module WxCore and header libwxcore.h"
  fi

  nlibs="$(find "$slice" -maxdepth 1 -name '*.a' | wc -l | tr -d '[:space:]')"
  [[ "$nlibs" == "1" ]] || fail "${id}: expected one static archive, found ${nlibs}"
  lib="$(find "$slice" -maxdepth 1 -name '*.a')"

  nm_out="$(nm -gU "$lib")"
  if ! printf '%s\n' "$nm_out" | grep -E ' [TDS] _?WxRunJSON$' >/dev/null; then
    fail "${id}: nm does not show WxRunJSON exported"
  fi
  if ! printf '%s\n' "$nm_out" | grep -E ' [TDS] _?WxFreeString$' >/dev/null; then
    fail "${id}: nm does not show WxFreeString exported"
  fi
  if ! printf '%s\n' "$nm_out" | grep -E ' [TDS] _?WxSetCacheDir$' >/dev/null; then
    fail "${id}: nm does not show WxSetCacheDir exported"
  fi
  echo "ok ${id}"
}

check_slice ios-arm64
check_slice ios-arm64-simulator

PRIV="${FW}/PrivacyInfo.xcprivacy"
[[ -f "$PRIV" ]] || fail "missing ${PRIV}"
plutil -lint "$PRIV" >/dev/null
track="$(plutil -extract NSPrivacyTracking raw -o - "$PRIV")"
[[ "$track" == "false" ]] || fail "WxCore privacy manifest must set NSPrivacyTracking false"
plutil -extract NSPrivacyAccessedAPITypes json -o - "$PRIV" | grep -q 'NSPrivacyAccessedAPICategoryFileTimestamp' \
  || fail "WxCore privacy manifest missing file-timestamp category"
plutil -extract NSPrivacyAccessedAPITypes json -o - "$PRIV" | grep -q 'C617.1' \
  || fail "WxCore privacy manifest missing C617.1"
plutil -extract NSPrivacyAccessedAPITypes json -o - "$PRIV" | grep -q 'NSPrivacyAccessedAPICategorySystemBootTime' \
  || fail "WxCore privacy manifest missing system-boot-time category"
plutil -extract NSPrivacyAccessedAPITypes json -o - "$PRIV" | grep -q '35F9.1' \
  || fail "WxCore privacy manifest missing 35F9.1"

echo "ios-framework-test passed"
