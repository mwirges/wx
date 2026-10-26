#!/usr/bin/env bash
# scripts/bump-version.sh — Bump version numbers across Go CLI, macOS app, iOS app, and README.
# The iOS app and the widget extension share one marketing version and one build number.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

VERSION_FILE="${ROOT_DIR}/VERSION"
GO_VERSION_FILE="${ROOT_DIR}/cmd/version.go"
PLIST_FILE="${ROOT_DIR}/macos/WxMac/Info.plist"
IOS_APP_PLIST="${ROOT_DIR}/ios/WxIOS/Info.plist"
IOS_WIDGET_PLIST="${ROOT_DIR}/ios/WxWidget/Info.plist"
README_FILE="${ROOT_DIR}/README.md"

set_short_version() {
  local file="$1"
  local version="$2"
  /usr/libexec/PlistBuddy -c "Set :CFBundleShortVersionString ${version}" "$file" 2>/dev/null || \
    /usr/libexec/PlistBuddy -c "Add :CFBundleShortVersionString string ${version}" "$file"
}

set_build_number() {
  local file="$1"
  local build="$2"
  /usr/libexec/PlistBuddy -c "Set :CFBundleVersion ${build}" "$file" 2>/dev/null || \
    /usr/libexec/PlistBuddy -c "Add :CFBundleVersion string ${build}" "$file"
}

CURRENT_VERSION="1.0.0"
if [[ -f "$VERSION_FILE" ]]; then
  CURRENT_VERSION="$(tr -d '[:space:]' < "$VERSION_FILE")"
fi

# Parse current version components (strip leading 'v' if present)
CLEAN_CURRENT="${CURRENT_VERSION#v}"
IFS='.' read -r MAJOR MINOR PATCH <<< "$CLEAN_CURRENT"
MAJOR="${MAJOR:-1}"
MINOR="${MINOR:-0}"
PATCH="${PATCH:-0}"

NEW_VERSION=""
INCREMENT_BUILD=true

# Parse arguments
while [[ $# -gt 0 ]]; do
  case "$1" in
    --version|-v)
      NEW_VERSION="${2#v}"
      shift 2
      ;;
    --patch|-p)
      NEW_VERSION="${MAJOR}.${MINOR}.$((PATCH + 1))"
      shift
      ;;
    --minor|-m)
      NEW_VERSION="${MAJOR}.$((MINOR + 1)).0"
      shift
      ;;
    --major|-M)
      NEW_VERSION="$((MAJOR + 1)).0.0"
      shift
      ;;
    --current|-c)
      NEW_VERSION="${CLEAN_CURRENT}"
      INCREMENT_BUILD=false
      shift
      ;;
    *)
      # If argument looks like a version string (e.g. 1.1.0 or v1.1.0)
      if [[ "$1" =~ ^v?[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        NEW_VERSION="${1#v}"
      else
        echo "error: unknown argument: $1" >&2
        echo "usage: $0 [--version X.Y.Z | --patch | --minor | --major | --current]" >&2
        exit 1
      fi
      shift
      ;;
  esac
done

# Default to minor bump if no argument provided
if [[ -z "$NEW_VERSION" ]]; then
  NEW_VERSION="${MAJOR}.$((MINOR + 1)).0"
fi

echo "==> Bumping version: ${CLEAN_CURRENT} → ${NEW_VERSION}"

# 1. Update VERSION file
echo "${NEW_VERSION}" > "$VERSION_FILE"

# 2. Update cmd/version.go
cat << EOF > "$GO_VERSION_FILE"
package cmd

// Version is the current semantic version of wx.
// Can be overridden at build time via -ldflags "-X github.com/mwirges/wx/cmd.Version=...".
var Version = "${NEW_VERSION}"
EOF

# 3. Update macos/WxMac/Info.plist
if [[ -f "$PLIST_FILE" ]]; then
  # Read current CFBundleVersion
  CURRENT_BUILD="$(/usr/libexec/PlistBuddy -c "Print :CFBundleVersion" "$PLIST_FILE" 2>/dev/null || echo "1")"
  if [[ "$INCREMENT_BUILD" == "true" ]]; then
    NEW_BUILD=$((CURRENT_BUILD + 1))
  else
    NEW_BUILD="${CURRENT_BUILD}"
  fi
  set_short_version "$PLIST_FILE" "${NEW_VERSION}"
  set_build_number "$PLIST_FILE" "${NEW_BUILD}"
fi

# 4. iOS app and widget: same marketing version, same build number.
for ios_plist in "$IOS_APP_PLIST" "$IOS_WIDGET_PLIST"; do
  if [[ ! -f "$ios_plist" ]]; then
    echo "error: missing ${ios_plist}" >&2
    exit 1
  fi
done
CURRENT_IOS_BUILD="$(/usr/libexec/PlistBuddy -c "Print :CFBundleVersion" "$IOS_APP_PLIST" 2>/dev/null || echo "0")"
if [[ "$INCREMENT_BUILD" == "true" ]]; then
  NEW_IOS_BUILD=$((CURRENT_IOS_BUILD + 1))
else
  NEW_IOS_BUILD="${CURRENT_IOS_BUILD}"
fi
for ios_plist in "$IOS_APP_PLIST" "$IOS_WIDGET_PLIST"; do
  set_short_version "$ios_plist" "${NEW_VERSION}"
  set_build_number "$ios_plist" "${NEW_IOS_BUILD}"
done
echo "==> iOS app and widget set to ${NEW_VERSION} (${NEW_IOS_BUILD})"

# 5. Update README.md release download links and badges
if [[ -f "$README_FILE" ]]; then
  # Replace download links for versioned dmg
  sed -i '' -E "s|wx-[0-9]+\.[0-9]+\.[0-9]+\.dmg|wx-${NEW_VERSION}.dmg|g" "$README_FILE"
  sed -i '' -E "s|releases/download/v[0-9]+\.[0-9]+\.[0-9]+/|releases/download/v${NEW_VERSION}/|g" "$README_FILE"
fi

echo "==> Version files updated successfully to ${NEW_VERSION}"
echo "${NEW_VERSION}"
