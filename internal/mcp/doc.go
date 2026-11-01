// Package mcp serves wx over the Model Context Protocol on loopback.
//
// wx mcp serve is part of the release binary (no build tag). The listener
// stays down unless mcp.enabled is set or the process is started with
// --enable. Bind is 127.0.0.1 only. Host and Origin are checked before
// /mcp or /healthz run.
//
// A mode-0600 pidfile and flock (mcp.pid, mcp.lock) beside the config file
// make a second serve refuse. This process never signals another pid.
// The Mac supervisor passes --parent-pid and --exit-on-stdin-close so the
// child exits when the app dies. Both watches are opt-in.
//
// When mcp.require_token is set, /mcp and /healthz both require
// Authorization: Bearer. Healthz is not an auth bypass. The token is read
// from a mode-0600 file and is never logged. A missing or wrong bearer is
// HTTP 401 with S6 code "unauthorized", before any tool runs.
//
// Weather tools share location.Resolve with the CLI, including lat,lon
// validation. Tool failures return isError with a JSON code and message.
// In-flight weather calls are capped per process. wx_server_info is not.
//
// wx_nowcast, wx_aqi, and wx_astro return the same JSON objects as
// wx nowcast --json, wx aqi --json, and wx astro --json.
// wx_radar defaults to metadata and never returns a multi-frame loop.
// include_image adds one frame. RadarImageMaxBytes (256 KiB) applies to
// the PNG and to its base64 form; over that, the code is payload_too_large.
package mcp
