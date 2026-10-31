package mcp

import (
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// guard rejects requests before they reach /mcp or /healthz.
// Host must be 127.0.0.1:<port> or localhost:<port>. A non-empty Origin
// must be http://127.0.0.1:<port> or http://localhost:<port>. Empty Origin
// is allowed (curl and the grok CLI send none). The handler does not copy
// Origin into any response header and does not set Access-Control-Allow-Origin.
func guard(port int, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !hostAllowed(r.Host, port) || !originAllowed(r.Header.Get("Origin"), port) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusForbidden)
			_, _ = io.WriteString(w, "forbidden\n")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func hostAllowed(hostport string, port int) bool {
	hostport = strings.TrimSpace(hostport)
	if hostport == "" || strings.ContainsAny(hostport, " \t") {
		return false
	}
	host, p, err := net.SplitHostPort(hostport)
	if err != nil || p != strconv.Itoa(port) {
		return false
	}
	switch strings.ToLower(host) {
	case "127.0.0.1", "localhost":
		return true
	default:
		return false
	}
}

func originAllowed(origin string, port int) bool {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" || u.Opaque != "" || u.User != nil {
		return false
	}
	if !strings.EqualFold(u.Scheme, "http") {
		return false
	}
	if u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return false
	}
	host, p, err := net.SplitHostPort(u.Host)
	if err != nil || p != strconv.Itoa(port) {
		return false
	}
	switch strings.ToLower(host) {
	case "127.0.0.1", "localhost":
		return true
	default:
		return false
	}
}
