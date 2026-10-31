// Package mcp serves wx over the Model Context Protocol on loopback.
//
// wx mcp serve is part of the release binary (no build tag). The listener
// stays down unless mcp.enabled is set or the process is started with
// --enable. Bind is 127.0.0.1 only. Host and Origin are checked before
// /mcp or /healthz run.
//
// Weather tools share location.Resolve with the CLI, including lat,lon
// validation. Tool failures return isError with a JSON code and message.
// In-flight weather calls are capped per process. wx_server_info is not.
package mcp
