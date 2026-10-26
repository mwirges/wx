#!/usr/bin/env bash
# Check an unsigned Release device .xcarchive. Does not export or upload.
# Usage: scripts/ios-archive-check.sh path/to/WxIOS.xcarchive
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ARCHIVE="${1:-}"

fail() {
  echo "ios-archive: $*" >&2
  exit 1
}

[[ -n "$ARCHIVE" && -d "$ARCHIVE" ]] || fail "pass the .xcarchive directory"

VERSION="$(tr -d '[:space:]' < "${ROOT}/VERSION")"
[[ -n "$VERSION" ]] || fail "VERSION is empty"

APP="$(find "${ARCHIVE}/Products/Applications" -maxdepth 1 -name '*.app' -print -quit)"
[[ -n "$APP" && -d "$APP" ]] || fail "no .app in Products/Applications"
APPEX="$(find "${APP}/PlugIns" -maxdepth 1 -name '*.appex' -print -quit 2>/dev/null || true)"
[[ -n "$APPEX" && -d "$APPEX" ]] || fail "no widget .appex in PlugIns"

INFO="${APP}/Info.plist"
WINFO="${APPEX}/Info.plist"
[[ -f "$INFO" && -f "$WINFO" ]] || fail "missing Info.plist"

raw() {
  plutil -extract "$1" raw -o - "$2"
}

json() {
  plutil -extract "$1" json -o - "$2"
}

app_ver="$(raw CFBundleShortVersionString "$INFO")"
wid_ver="$(raw CFBundleShortVersionString "$WINFO")"
app_build="$(raw CFBundleVersion "$INFO")"
wid_build="$(raw CFBundleVersion "$WINFO")"
[[ "$app_ver" == "$VERSION" ]] || fail "app version ${app_ver} != VERSION ${VERSION}"
[[ "$wid_ver" == "$app_ver" ]] || fail "widget version ${wid_ver} != app ${app_ver}"
[[ "$wid_build" == "$app_build" ]] || fail "widget build ${wid_build} != app ${app_build}"
[[ "$app_build" =~ ^[0-9]+(\.[0-9]+)*$ ]] || fail "CFBundleVersion ${app_build} is not a numeric build"

[[ "$(raw CFBundleIdentifier "$INFO")" == "com.mwirges.wx.ios" ]] || fail "app bundle id"
[[ "$(raw CFBundleIdentifier "$WINFO")" == "com.mwirges.wx.ios.widget" ]] || fail "widget bundle id"
[[ "$(raw CFBundleExecutable "$INFO")" == "wx" ]] || fail "app executable name"
[[ "$(raw CFBundleExecutable "$WINFO")" == "WxWidget" ]] || fail "widget executable name"
[[ "$(raw CFBundlePackageType "$INFO")" == "APPL" ]] || fail "app package type"
[[ "$(raw CFBundlePackageType "$WINFO")" == "XPC!" ]] || fail "widget package type"
[[ "$(raw CFBundleIconName "$INFO")" == "AppIcon" ]] || fail "CFBundleIconName"
[[ "$(raw MinimumOSVersion "$INFO")" == "17.0" ]] || fail "app MinimumOSVersion"
[[ "$(raw MinimumOSVersion "$WINFO")" == "17.0" ]] || fail "widget MinimumOSVersion"
[[ "$(raw DTPlatformName "$INFO")" == "iphoneos" ]] || fail "app DTPlatformName is not iphoneos"
[[ "$(raw DTPlatformName "$WINFO")" == "iphoneos" ]] || fail "widget DTPlatformName is not iphoneos"
[[ "$(raw ITSAppUsesNonExemptEncryption "$INFO")" == "false" ]] || fail "app encryption flag"
[[ "$(raw ITSAppUsesNonExemptEncryption "$WINFO")" == "false" ]] || fail "widget encryption flag"
[[ "$(raw NSSupportsLiveActivities "$INFO")" == "true" ]] || fail "NSSupportsLiveActivities"
raw NSLocationWhenInUseUsageDescription "$INFO" >/dev/null || fail "missing When In Use purpose string"
if raw NSLocationAlwaysAndWhenInUseUsageDescription "$INFO" >/dev/null 2>&1; then
  fail "NSLocationAlwaysAndWhenInUseUsageDescription is set"
fi
if raw NSLocationAlwaysUsageDescription "$INFO" >/dev/null 2>&1; then
  fail "NSLocationAlwaysUsageDescription is set"
fi

family="$(json UIDeviceFamily "$INFO" | tr -d '[:space:]')"
wfamily="$(json UIDeviceFamily "$WINFO" | tr -d '[:space:]')"
[[ "$family" == "[1,2]" ]] || fail "app UIDeviceFamily ${family}"
[[ "$wfamily" == "[1,2]" ]] || fail "widget UIDeviceFamily ${wfamily}"

json UILaunchScreen "$INFO" >/dev/null || fail "missing UILaunchScreen"
json UIApplicationSceneManifest "$INFO" >/dev/null || fail "missing UIApplicationSceneManifest"

phone="$(json UISupportedInterfaceOrientations "$INFO" | tr -d '[:space:]')"
for orient in UIInterfaceOrientationPortrait UIInterfaceOrientationLandscapeLeft UIInterfaceOrientationLandscapeRight; do
  [[ "$phone" == *"$orient"* ]] || fail "iPhone missing ${orient}"
done
pad="$(json 'UISupportedInterfaceOrientations~ipad' "$INFO" | tr -d '[:space:]')"
for orient in UIInterfaceOrientationPortrait UIInterfaceOrientationPortraitUpsideDown UIInterfaceOrientationLandscapeLeft UIInterfaceOrientationLandscapeRight; do
  [[ "$pad" == *"$orient"* ]] || fail "iPad missing ${orient}"
done
if json UIRequiresFullScreen "$INFO" >/dev/null 2>&1; then
  full="$(raw UIRequiresFullScreen "$INFO" || true)"
  [[ "$full" == "false" ]] || fail "UIRequiresFullScreen blocks iPad multitasking"
fi

point="$(raw NSExtension.NSExtensionPointIdentifier "$WINFO")"
[[ "$point" == "com.apple.widgetkit-extension" ]] || fail "widget extension point ${point}"

[[ -x "${APP}/wx" || -f "${APP}/wx" ]] || fail "missing app executable"
[[ -f "${APPEX}/WxWidget" ]] || fail "missing widget executable"

check_mach() {
  local bin="$1"
  local label="$2"
  local archs platform
  archs="$(lipo -archs "$bin" | tr -d '[:space:]')"
  [[ "$archs" == "arm64" ]] || fail "${label} archs ${archs}, want arm64 only"
  platform="$(xcrun vtool -show-build "$bin" 2>/dev/null || true)"
  [[ "$platform" == *"platform IOS"* ]] || fail "${label} is not platform IOS"
  if [[ "$platform" == *"IOSSIMULATOR"* || "$platform" == *"platform IOSSIM"* ]]; then
    fail "${label} contains a simulator platform"
  fi
  [[ "$platform" == *"minos 17.0"* ]] || fail "${label} minos is not 17.0"
  if otool -l "$bin" | grep -q 'sectname __LLVM'; then
    fail "${label} contains bitcode"
  fi
  if codesign -dv "$bin" 2>&1 | grep -E 'TeamIdentifier=[A-Z0-9]{10}'; then
    fail "${label} is signed with a team id"
  fi
}

check_mach "${APP}/wx" "wx"
check_mach "${APPEX}/WxWidget" "WxWidget"

wxcore_hit="$(find "$APP" \( -name 'WxCore*' -o -name 'libwxcore*' \) -print -quit)"
[[ -z "$wxcore_hit" ]] || fail "WxCore is embedded in the app (${wxcore_hit}); it is a static archive and must be linked, not copied"
wxcore_widget="$(find "$APPEX" \( -name 'WxCore*' -o -name 'libwxcore*' \) -print -quit)"
[[ -z "$wxcore_widget" ]] || fail "widget contains WxCore (${wxcore_widget})"

# Static WxCore has no framework bundle. An embedded framework still needs versions.
if [[ -d "${APP}/Frameworks" ]]; then
  while IFS= read -r fwinfo; do
    raw CFBundleShortVersionString "$fwinfo" >/dev/null || fail "missing CFBundleShortVersionString in ${fwinfo}"
    raw MinimumOSVersion "$fwinfo" >/dev/null || fail "missing MinimumOSVersion in ${fwinfo}"
  done < <(find "${APP}/Frameworks" -name Info.plist)
fi

ent_hit="$(find "$APP" -name '*.entitlements' -print -quit)"
[[ -z "$ent_hit" ]] || fail "entitlements file copied into the bundle (${ent_hit})"

[[ -f "${APP}/Assets.car" ]] || fail "missing compiled asset catalog"
[[ -s "${APP}/Assets.car" ]] || fail "Assets.car is empty"
car_info="$(xcrun --sdk iphoneos assetutil --info "${APP}/Assets.car")"
printf '%s\n' "$car_info" | grep -q 'AppIcon' || fail "Assets.car has no AppIcon"
printf '%s\n' "$car_info" | grep -q '1024' || fail "Assets.car has no 1024 icon"

icon="${ROOT}/ios/WxIOS/Assets.xcassets/AppIcon.appiconset/AppIcon-1024.png"
[[ -f "$icon" ]] || fail "missing marketing icon source"
alpha="$(sips -g hasAlpha "$icon" | awk '/hasAlpha/ {print $2}')"
[[ "$alpha" == "no" ]] || fail "marketing icon has an alpha channel"
width="$(sips -g pixelWidth "$icon" | awk '/pixelWidth/ {print $2}')"
height="$(sips -g pixelHeight "$icon" | awk '/pixelHeight/ {print $2}')"
[[ "$width" == "1024" && "$height" == "1024" ]] || fail "marketing icon is ${width}x${height}"

[[ -f "${APP}/PrivacyInfo.xcprivacy" ]] || fail "missing app privacy manifest"
[[ -f "${APPEX}/PrivacyInfo.xcprivacy" ]] || fail "missing widget privacy manifest"
plutil -lint "${APP}/PrivacyInfo.xcprivacy" >/dev/null
plutil -lint "${APPEX}/PrivacyInfo.xcprivacy" >/dev/null
[[ "$(raw NSPrivacyTracking "${APP}/PrivacyInfo.xcprivacy")" == "false" ]] || fail "app tracking flag"
[[ "$(raw NSPrivacyTracking "${APPEX}/PrivacyInfo.xcprivacy")" == "false" ]] || fail "widget tracking flag"

app_privacy="$(json NSPrivacyCollectedDataTypes "${APP}/PrivacyInfo.xcprivacy")"
printf '%s\n' "$app_privacy" | grep -q 'NSPrivacyCollectedDataTypePreciseLocation' || fail "app manifest missing precise location"
printf '%s\n' "$app_privacy" | grep -q 'NSPrivacyCollectedDataTypeCoarseLocation' || fail "app manifest missing coarse location"
printf '%s\n' "$app_privacy" | grep -q 'NSPrivacyCollectedDataTypePurposeAppFunctionality' || fail "location purpose"
# Both location types are not linked and not used for tracking.
python3 - "$app_privacy" << 'PY'
import json, sys
rows = json.loads(sys.argv[1])
want = {
    "NSPrivacyCollectedDataTypePreciseLocation",
    "NSPrivacyCollectedDataTypeCoarseLocation",
}
seen = set()
for row in rows:
    kind = row.get("NSPrivacyCollectedDataType")
    if kind not in want:
        continue
    seen.add(kind)
    if row.get("NSPrivacyCollectedDataTypeLinked") is not False:
        raise SystemExit(f"ios-archive: {kind} is linked")
    if row.get("NSPrivacyCollectedDataTypeTracking") is not False:
        raise SystemExit(f"ios-archive: {kind} is tracking")
    purposes = row.get("NSPrivacyCollectedDataTypePurposes") or []
    if purposes != ["NSPrivacyCollectedDataTypePurposeAppFunctionality"]:
        raise SystemExit(f"ios-archive: {kind} purposes {purposes}")
missing = want - seen
if missing:
    raise SystemExit(f"ios-archive: missing collected types {sorted(missing)}")
PY

widget_collected="$(json NSPrivacyCollectedDataTypes "${APPEX}/PrivacyInfo.xcprivacy" | tr -d '[:space:]')"
[[ "$widget_collected" == "[]" ]] || fail "widget collects data; it only reads the app group cache"

# Required-reason symbols the App Store scanner looks for in the binary.
imports_of() {
  {
    nm -u "$1" 2>/dev/null | awk '{print $NF}'
    nm -m "$1" 2>/dev/null | awk '/undefined/ {print $NF}'
  } | sed 's/^_//' | sort -u
}

require_reason() {
  local bin="$1"
  local manifest="$2"
  local category="$3"
  shift 3
  local imports hit sym
  imports="$(imports_of "$bin")"
  hit=""
  for sym in "$@"; do
    if printf '%s\n' "$imports" | grep -qx "$sym"; then
      hit="${hit} ${sym}"
    fi
  done
  if [[ -n "$hit" ]]; then
    local declared
    declared="$(json NSPrivacyAccessedAPITypes "$manifest" 2>/dev/null || echo '[]')"
    if ! printf '%s\n' "$declared" | grep -q "$category"; then
      fail "$(basename "$bin") references${hit} but privacy manifest has no ${category}"
    fi
  fi
}

require_reason "${APP}/wx" "${APP}/PrivacyInfo.xcprivacy" \
  NSPrivacyAccessedAPICategoryFileTimestamp \
  stat lstat fstat fstatat getattrlist getattrlistbulk fgetattrlist getattrlistat
require_reason "${APP}/wx" "${APP}/PrivacyInfo.xcprivacy" \
  NSPrivacyAccessedAPICategorySystemBootTime \
  mach_absolute_time
require_reason "${APP}/wx" "${APP}/PrivacyInfo.xcprivacy" \
  NSPrivacyAccessedAPICategoryDiskSpace \
  statfs fstatfs statvfs fstatvfs
require_reason "${APP}/wx" "${APP}/PrivacyInfo.xcprivacy" \
  NSPrivacyAccessedAPICategoryUserDefaults \
  NSUserDefaults
require_reason "${APPEX}/WxWidget" "${APPEX}/PrivacyInfo.xcprivacy" \
  NSPrivacyAccessedAPICategoryFileTimestamp \
  stat lstat fstat fstatat getattrlist getattrlistbulk fgetattrlist getattrlistat
require_reason "${APPEX}/WxWidget" "${APPEX}/PrivacyInfo.xcprivacy" \
  NSPrivacyAccessedAPICategorySystemBootTime \
  mach_absolute_time
require_reason "${APPEX}/WxWidget" "${APPEX}/PrivacyInfo.xcprivacy" \
  NSPrivacyAccessedAPICategoryDiskSpace \
  statfs fstatfs statvfs fstatvfs
require_reason "${APPEX}/WxWidget" "${APPEX}/PrivacyInfo.xcprivacy" \
  NSPrivacyAccessedAPICategoryUserDefaults \
  NSUserDefaults

# Swift may reference the ObjC class without a plain NSUserDefaults symbol.
if nm -m "${APP}/wx" "${APPEX}/WxWidget" 2>/dev/null | grep -q 'NSUserDefaults'; then
  for pair in "${APP}/wx|${APP}/PrivacyInfo.xcprivacy" "${APPEX}/WxWidget|${APPEX}/PrivacyInfo.xcprivacy"; do
    bin="${pair%%|*}"
    manifest="${pair#*|}"
    if nm -m "$bin" 2>/dev/null | grep -q 'NSUserDefaults'; then
      declared="$(json NSPrivacyAccessedAPITypes "$manifest")"
      printf '%s\n' "$declared" | grep -q 'NSPrivacyAccessedAPICategoryUserDefaults' \
        || fail "$(basename "$bin") references NSUserDefaults without a UserDefaults reason"
    fi
  done
fi

shopt -s nullglob
dsyms=("${ARCHIVE}/dSYMs/"*.dSYM)
[[ "${#dsyms[@]}" -ge 1 ]] || fail "no dSYM in ${ARCHIVE}/dSYMs"
found_app=0
found_widget=0
for dsym in "${dsyms[@]}"; do
  base="$(basename "$dsym")"
  [[ "$base" == *wx.app* ]] && found_app=1
  [[ "$base" == *WxWidget* ]] && found_widget=1
  dwarf="$(find "$dsym" \( -name 'wx' -o -name 'WxWidget' \) -type f -print -quit)"
  [[ -n "$dwarf" ]] || fail "dSYM ${base} has no DWARF binary"
done
[[ "$found_app" == 1 ]] || fail "missing wx.app dSYM"
[[ "$found_widget" == 1 ]] || fail "missing WxWidget dSYM"

# Source files the unsigned archive does not seal, but the signed upload needs.
# plutil -extract splits on '.', so a key like com.apple.security.application-groups
# cannot be addressed that way. PlistBuddy colon paths keep the dotted key intact.
for ent in "${ROOT}/ios/WxIOS/WxIOS.entitlements" "${ROOT}/ios/WxWidget/WxWidget.entitlements"; do
  [[ -f "$ent" ]] || fail "missing ${ent}"
  group="$(/usr/libexec/PlistBuddy -c 'Print :com.apple.security.application-groups:0' "$ent")"
  [[ "$group" == "group.com.mwirges.wx" ]] || fail "${ent} app group is ${group}"
done

opts="${ROOT}/ios/ExportOptions.plist"
[[ -f "$opts" ]] || fail "missing ExportOptions.plist"
[[ "$(raw method "$opts")" == "app-store-connect" ]] || fail "export method"
[[ "$(raw signingStyle "$opts")" == "automatic" ]] || fail "export signingStyle"
[[ "$(raw teamID "$opts")" == '$(WX_TEAM_ID)' ]] || fail "export teamID must stay the placeholder"
[[ "$(raw destination "$opts")" == "export" ]] || fail "export destination"
if grep -E 'DEVELOPMENT_TEAM: *"[A-Z0-9]{10}"' "${ROOT}/ios/project.yml" >/dev/null; then
  fail "project.yml contains a real DEVELOPMENT_TEAM"
fi

arch_plist="${ARCHIVE}/Info.plist"
[[ -f "$arch_plist" ]] || fail "missing archive Info.plist"
[[ "$(raw ApplicationProperties.CFBundleShortVersionString "$arch_plist")" == "$VERSION" ]] || fail "archive marketing version"
[[ "$(raw ApplicationProperties.CFBundleVersion "$arch_plist")" == "$app_build" ]] || fail "archive build number"
[[ "$(raw ApplicationProperties.CFBundleIdentifier "$arch_plist")" == "com.mwirges.wx.ios" ]] || fail "archive bundle id"

echo "ios archive ok ${app_ver} (${app_build}) device arm64 widget ${wid_ver} (${wid_build})"
