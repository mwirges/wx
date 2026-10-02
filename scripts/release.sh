#!/usr/bin/env bash
# scripts/release.sh — End-to-end automated release pipeline for wx
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${ROOT_DIR}"

DRY_RUN=false
SKIP_TESTS=false
ALLOW_DIRTY=false
NOTARIZE="${NOTARIZE:-true}"
BUMP_ARGS=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dry-run)
      DRY_RUN=true
      NOTARIZE=false
      shift
      ;;
    --skip-tests)
      SKIP_TESTS=true
      shift
      ;;
    --skip-notarize|--no-notarize)
      NOTARIZE=false
      shift
      ;;
    --allow-dirty)
      ALLOW_DIRTY=true
      shift
      ;;
    --version|-v)
      BUMP_ARGS+=(--version "$2")
      shift 2
      ;;
    --patch|-p|--minor|-m|--major|-M|--current|-c)
      BUMP_ARGS+=("$1")
      shift
      ;;
    *)
      if [[ "$1" =~ ^v?[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        BUMP_ARGS+=(--version "${1#v}")
      else
        echo "error: unknown argument: $1" >&2
        echo "usage: $0 [--version X.Y.Z | --patch | --minor | --major | --current] [--dry-run] [--skip-tests]" >&2
        exit 1
      fi
      shift
      ;;
  esac
done

echo "============================================================"
echo "          wx Automated Release Pipeline                     "
echo "============================================================"

# 1. Preflight checks
echo "==> Running preflight checks..."

if ! command -v git >/dev/null 2>&1; then
  echo "error: git is required but not installed." >&2
  exit 1
fi

if ! command -v gh >/dev/null 2>&1; then
  echo "error: GitHub CLI (gh) is required but not installed." >&2
  exit 1
fi

if ! gh auth status >/dev/null 2>&1; then
  echo "error: GitHub CLI is not authenticated. Please run 'gh auth login'." >&2
  exit 1
fi

# Check git status
if [[ "$ALLOW_DIRTY" != "true" ]]; then
  # Allow uncommitted changes only if they are related to version/doc prep
  DIRTY_FILES="$(git status --porcelain 2>/dev/null | grep -v -E "VERSION|cmd/version.go|Info.plist|AppDelegate.swift|README.md" || true)"
  if [[ -n "$DIRTY_FILES" ]]; then
    echo "error: Working tree has unstaged or untracked changes:" >&2
    git status --short
    echo "Please commit or stash your changes before releasing." >&2
    exit 1
  fi
fi

# Capture changelog from previous tag
PREV_TAG="$(git describe --tags --abbrev=0 2>/dev/null || echo '')"
CHANGELOG=""
if [[ -n "${PREV_TAG}" ]]; then
  CHANGELOG="$(git log "${PREV_TAG}..HEAD" --no-merges --pretty=format:"- %s (%h)")"
fi

# 2. Bump version
echo "==> Synchronizing version metadata..."
VERSION_OUTPUT="$("${SCRIPT_DIR}/bump-version.sh" "${BUMP_ARGS[@]}")"
TARGET_VERSION="$(echo "${VERSION_OUTPUT}" | tail -n 1 | tr -d '[:space:]')"
TAG_NAME="v${TARGET_VERSION}"

echo "==> Target release version: ${TARGET_VERSION} (tag: ${TAG_NAME})"

# Check if tag already exists
if git rev-parse "${TAG_NAME}" >/dev/null 2>&1; then
  echo "error: Git tag ${TAG_NAME} already exists." >&2
  exit 1
fi

# 3. Verification & Tests
if [[ "$SKIP_TESTS" != "true" ]]; then
  echo "==> Running tests and linter (make test && make vet)..."
  make test
  make vet
  echo "==> All test suites passed."
fi

# 4. Build macOS DMG Installer
echo "==> Building macOS release DMG..."
if [[ "$NOTARIZE" == "true" ]]; then
  echo "==> Apple Notarization enabled (NOTARIZE=1)..."
  NOTARIZE=1 make dmg
else
  echo "==> Note: Notarization skipped."
  make dmg
fi

if [[ ! -f "build/wx.dmg" ]]; then
  echo "error: build/wx.dmg was not generated." >&2
  exit 1
fi

# Create versioned copy of DMG (inherits stapled ticket from wx.dmg)
VERSIONED_DMG="build/wx-${TARGET_VERSION}.dmg"
cp -f "build/wx.dmg" "${VERSIONED_DMG}"

if [[ "$NOTARIZE" == "true" ]]; then
  echo "==> Verifying Gatekeeper approval on packaged DMGs..."
  spctl -a -vvv -t install "build/wx.dmg"
  spctl -a -vvv -t install "${VERSIONED_DMG}"
fi

echo "==> Packaged artifacts:"
ls -lh "build/wx.dmg" "${VERSIONED_DMG}"

# Dry run exit point
if [[ "$DRY_RUN" == "true" ]]; then
  echo "==> [DRY RUN] Release prepared successfully. No git commit, tag, or GitHub release was created."
  exit 0
fi

# 5. Git Commit & Tag
echo "==> Committing release version files..."
git add VERSION cmd/version.go macos/WxMac/Info.plist macos/WxMac/AppDelegate.swift README.md
if ! git diff --cached --quiet; then
  git commit -m "release: ${TAG_NAME}"
else
  echo "Note: Version files already committed."
fi

echo "==> Tagging release ${TAG_NAME}..."
git tag -a "${TAG_NAME}" -m "wx release ${TAG_NAME}"

# 6. Push commit and tag to origin
echo "==> Pushing commit and tag to remote..."
git push origin HEAD
git push origin "${TAG_NAME}"

# 7. Create GitHub Release
echo "==> Publishing GitHub Release ${TAG_NAME} with attached DMG..."
RELEASE_NOTES_FILE="$(mktemp /tmp/wx-release-notes.XXXXXX)"
trap 'rm -f "${RELEASE_NOTES_FILE}"' EXIT

cat << EOF > "${RELEASE_NOTES_FILE}"
## wx ${TAG_NAME}

A terminal weather suite and native macOS app for US (NWS/MRMS) and global (Open-Meteo) weather intelligence.

### What's Changed
${CHANGELOG:-No changes recorded.}

### Artifacts Included
- **\`wx.dmg\`**: Universal drag-and-drop macOS disk image installer.
- **\`wx-${TARGET_VERSION}.dmg\`**: Version-pinned macOS disk image installer.
- **Embedded Engine**: Both DMGs bundle the native macOS menu bar HUD, Command Console (\`WX.DESK\`), and the compiled Go CLI binary inside \`wx.app/Contents/MacOS/wx-cli\`.

### Highlights
- **10 Core Meteorological Intelligence Systems**:
  - Quantitative precipitation nowcasting (\`wx nowcast\` / \`wx precip\`)
  - NOAA 30-year climate normals & daily records (\`wx climate\` / \`wx records\`)
  - Upper-air thermodynamic soundings (\`wx sounding\` / \`wx cape\`)
  - NHC tropical cyclone tracking (\`wx tropics\` / \`wx nhc\`)
  - SPC convective risk & severe storm outlooks (\`wx spc\` / \`wx chase\`)
  - US EPA AQI & chemical atmospheric telemetry (\`wx aqi\` / \`wx air\`)
  - 100% offline astronomical ephemeris & solar arc HUD (\`wx astro\` / \`wx ephem\`)
  - NWS Area Forecast Discussions & CPC synoptic outlooks (\`wx outlook\`)
  - Multi-day historical surface observations (\`wx history\`)
  - High-resolution Doppler radar with animated GIF export (\`wx radar --save-gif\`)
- **macOS Native App Enhancements**:
  - 8 Specialized Tactical Tabs (\`Tactical\`, \`Surface\`, \`Radar\`, \`CPC Outlooks\`, \`Storm Chase\`, \`Climate\`, \`Tropics\`, \`Grid\`)
  - 7 Customizable Menu Bar HUD Display Formats (\`standard\`, \`compact\`, \`tactical\`, \`conditions\`, \`ephemeris\`, \`icon\`, \`minimal\`)
  - Real-time severe weather alert notification pulses and vector Apple Maps radar reflectivity.
EOF

gh release create "${TAG_NAME}" \
  "build/wx.dmg" \
  "${VERSIONED_DMG}" \
  --title "wx ${TAG_NAME}" \
  --notes-file "${RELEASE_NOTES_FILE}"

echo "============================================================"
echo "  Successfully released wx ${TAG_NAME}!"
echo "  Release URL: https://github.com/mwirges/wx/releases/tag/${TAG_NAME}"
echo "============================================================"
