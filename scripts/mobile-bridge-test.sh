#!/usr/bin/env bash
# Build pkg/mobile as a C archive, call it from C, and compare with wx --json.
# HTTP comes from pkg/mobile/testdata/stub.go via WX_STUB_ORIGIN. No live NWS.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
mkdir -p build

echo "==> wx CLI"
make build

echo "==> c-archive"
CGO_ENABLED=1 go build -buildmode=c-archive -o build/libwxmobile.a ./pkg/mobile

echo "==> C harness"
# Go's c-archive needs the host's system libraries. CoreFoundation and
# Security are Darwin-only; Linux wants pthread and dl.
cc_cmd=(cc -I build -o build/mobile-bridge-harness \
  pkg/mobile/testdata/harness.c build/libwxmobile.a)
case "$(uname -s)" in
  Darwin)
    cc_cmd+=(-framework CoreFoundation -framework Security)
    ;;
  Linux)
    cc_cmd+=(-lpthread -ldl)
    ;;
esac
"${cc_cmd[@]}"

echo "==> stub server"
go build -o build/wx-mobile-stub pkg/mobile/testdata/stub.go
go build -o build/wx-mobile-compare pkg/mobile/testdata/compare.go

ADDR_FILE="$(mktemp)"
TMP_HOME="$(mktemp -d)"
STUB_PID=""
cleanup() {
  if [[ -n "$STUB_PID" ]]; then
    kill "$STUB_PID" 2>/dev/null || true
    wait "$STUB_PID" 2>/dev/null || true
  fi
  rm -rf "$TMP_HOME" "$ADDR_FILE"
}
trap cleanup EXIT

./build/wx-mobile-stub >"$ADDR_FILE" &
STUB_PID=$!
for _ in $(seq 1 50); do
  if [[ -s "$ADDR_FILE" ]]; then
    break
  fi
  sleep 0.1
done
ADDR="$(head -n 1 "$ADDR_FILE" | tr -d '[:space:]')"
if [[ -z "$ADDR" ]]; then
  echo "stub server did not start" >&2
  exit 1
fi

export WX_STUB_ORIGIN="http://${ADDR}"
export HOME="$TMP_HOME"
WX="$ROOT/build/wx"
HARNESS="$ROOT/build/mobile-bridge-harness"
COMPARE="$ROOT/build/wx-mobile-compare"

run_case() {
  local name="$1"
  local args_json="$2"
  shift 2
  local cli_out bridge_out cli_err bridge_err
  cli_out="$(mktemp)"
  bridge_out="$(mktemp)"
  cli_err="$(mktemp)"
  bridge_err="$(mktemp)"
  if ! "$WX" "$@" >"$cli_out" 2>"$cli_err"; then
    echo "CLI failed for ${name}: $*" >&2
    cat "$cli_err" >&2
    echo "--- stdout ---" >&2
    cat "$cli_out" >&2
    exit 1
  fi
  if ! "$HARNESS" "$args_json" >"$bridge_out" 2>"$bridge_err"; then
    echo "harness crashed for ${name}" >&2
    cat "$bridge_err" >&2
    echo "--- stdout ---" >&2
    cat "$bridge_out" >&2
    exit 1
  fi
  if ! grep -q "New Haven" "$bridge_out"; then
    echo "${name}: location was not kept as one argument" >&2
    cat "$bridge_out" >&2
    exit 1
  fi
  if ! "$COMPARE" "$cli_out" "$bridge_out"; then
    echo "${name}: bridge JSON does not match wx $*" >&2
    if [[ -s "$cli_err" ]]; then
      echo "--- cli stderr ---" >&2
      cat "$cli_err" >&2
    fi
    if [[ -s "$bridge_err" ]]; then
      echo "--- bridge stderr ---" >&2
      cat "$bridge_err" >&2
    fi
    exit 1
  fi
  rm -f "$cli_out" "$bridge_out" "$cli_err" "$bridge_err"
  echo "ok ${name}"
}

run_case forecast '["forecast","--no-cache","New Haven, IN"]' \
  --forecast --json --no-cache "New Haven, IN"
run_case alerts '["alerts","--no-cache","New Haven, IN"]' \
  --alerts --json --no-cache "New Haven, IN"
run_case hourly '["hourly","--no-cache","New Haven, IN"]' \
  hourly --json --no-cache "New Haven, IN"
run_case astro '["astro","--date","2026-06-21","New Haven, IN"]' \
  astro --json --date 2026-06-21 "New Haven, IN"
run_case aqi '["aqi","--no-cache","New Haven, IN"]' \
  aqi --json --no-cache "New Haven, IN"
run_case spc '["spc","--no-cache","New Haven, IN"]' \
  spc --json --no-cache "New Haven, IN"
run_case nowcast '["nowcast","New Haven, IN"]' \
  nowcast --json "New Haven, IN"
run_case radar '["radar","--no-cache","New Haven, IN"]' \
  radar --json --no-cache "New Haven, IN"
run_case climate '["climate","--date","2026-06-21","New Haven, IN"]' \
  climate --json --date 2026-06-21 "New Haven, IN"
run_case tropics '["tropics","New Haven, IN"]' \
  tropics --json "New Haven, IN"
run_case spc-json-after '["spc","New Haven, IN","--json"]' \
  spc --json "New Haven, IN"
run_case aqi-place-only '["aqi","New Haven, IN"]' \
  aqi --json "New Haven, IN"

expect_error() {
  local name="$1"
  local args_json="$2"
  local code="$3"
  local out
  if ! out="$("$HARNESS" "$args_json")"; then
    echo "harness crashed for error case ${name}" >&2
    exit 1
  fi
  if ! printf '%s' "$out" | grep -q "\"code\":\"${code}\""; then
    echo "${name}: want code ${code}, got:" >&2
    printf '%s\n' "$out" >&2
    exit 1
  fi
  echo "ok ${name}"
}

expect_error unknown '["chase","New Haven, IN"]' unknown_command
expect_error bad-json 'this is not json' invalid_json

echo "mobile-bridge-test passed"
