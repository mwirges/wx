package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func testServer(t *testing.T) *Server {
	t.Helper()
	s, err := New(Options{
		Enabled:         true,
		Port:            DefaultPort,
		Version:         "1.2.0-test",
		DefaultLocation: "New Haven, IN",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return s
}

func doReq(t *testing.T, h http.Handler, method, path, host, origin, body string) *httptest.ResponseRecorder {
	t.Helper()
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, "http://127.0.0.1:9742"+path, rdr)
	req.Host = host
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if acao := rr.Header().Get("Access-Control-Allow-Origin"); acao != "" {
		t.Errorf("Access-Control-Allow-Origin = %q, want none", acao)
	}
	return rr
}

func TestGuardRejectsHostAndOrigin(t *testing.T) {
	s := testServer(t)
	h := s.Handler()

	cases := []struct {
		name   string
		method string
		path   string
		host   string
		origin string
		body   string
	}{
		{"missing host healthz", http.MethodGet, "/healthz", "", "", ""},
		{"missing host mcp", http.MethodPost, "/mcp", "", "", `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`},
		{"evil host healthz", http.MethodGet, "/healthz", "evil.example", "", ""},
		{"evil host with port", http.MethodGet, "/healthz", "evil.example:9742", "", ""},
		{"evil host mcp", http.MethodPost, "/mcp", "evil.example", "", `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`},
		{"wrong port", http.MethodGet, "/healthz", "127.0.0.1:80", "", ""},
		{"bare ip", http.MethodGet, "/healthz", "127.0.0.1", "", ""},
		{"ipv6 loopback", http.MethodGet, "/healthz", "[::1]:9742", "", ""},
		{"wildcard", http.MethodGet, "/healthz", "0.0.0.0:9742", "", ""},
		{"evil origin healthz", http.MethodGet, "/healthz", "127.0.0.1:9742", "http://evil.example", ""},
		{"evil origin mcp", http.MethodPost, "/mcp", "127.0.0.1:9742", "http://evil.example", `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`},
		{"null origin", http.MethodGet, "/healthz", "127.0.0.1:9742", "null", ""},
		{"https origin", http.MethodGet, "/healthz", "localhost:9742", "https://127.0.0.1:9742", ""},
		{"origin wrong port", http.MethodGet, "/healthz", "127.0.0.1:9742", "http://127.0.0.1:80", ""},
		{"origin with path", http.MethodGet, "/healthz", "127.0.0.1:9742", "http://127.0.0.1:9742/mcp", ""},
		{"origin userinfo", http.MethodGet, "/healthz", "127.0.0.1:9742", "http://user@127.0.0.1:9742", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := s.Hits()
			rr := doReq(t, h, tc.method, tc.path, tc.host, tc.origin, tc.body)
			if rr.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want 403; body %q", rr.Code, rr.Body.String())
			}
			if rr.Body.String() != "forbidden\n" {
				t.Fatalf("body = %q", rr.Body.String())
			}
			if strings.Contains(rr.Body.String(), "evil") {
				t.Fatalf("response reflected host: %q", rr.Body.String())
			}
			if tc.origin != "" && strings.Contains(rr.Body.String(), tc.origin) {
				t.Fatalf("response reflected origin: %q", rr.Body.String())
			}
			if strings.Contains(rr.Body.String(), "1.2.0-test") {
				t.Fatal("rejected request returned the version body")
			}
			if s.Hits() != before {
				t.Fatalf("hits %d -> %d; guard ran a handler", before, s.Hits())
			}
		})
	}
}

func TestGuardAllowsLoopback(t *testing.T) {
	s := testServer(t)
	h := s.Handler()

	cases := []struct {
		name   string
		host   string
		origin string
	}{
		{"ip empty origin", "127.0.0.1:9742", ""},
		{"localhost empty origin", "localhost:9742", ""},
		{"mixed case localhost", "LocalHost:9742", ""},
		{"ip origin", "127.0.0.1:9742", "http://127.0.0.1:9742"},
		{"localhost origin", "localhost:9742", "http://localhost:9742"},
		{"cross loopback names", "127.0.0.1:9742", "http://localhost:9742"},
		{"origin case", "127.0.0.1:9742", "HTTP://127.0.0.1:9742"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := s.Hits()
			rr := doReq(t, h, http.MethodGet, "/healthz", tc.host, tc.origin, "")
			if rr.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200; body %q", rr.Code, rr.Body.String())
			}
			if rr.Body.String() != "1.2.0-test\n" {
				t.Fatalf("body = %q", rr.Body.String())
			}
			if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "text/plain") {
				t.Fatalf("content-type = %q", ct)
			}
			if s.Hits() != before+1 {
				t.Fatalf("hits = %d, want %d", s.Hits(), before+1)
			}
		})
	}
}

func TestMCPDispatchAfterGuard(t *testing.T) {
	s := testServer(t)
	h := s.Handler()
	// A disallowed content type still passes the guard and reaches the MCP
	// handler, which rejects it. A valid initialize can hold the response
	// open for SSE, so this case stays a fast non-403.
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:9742/mcp", strings.NewReader("nope"))
	req.Host = "127.0.0.1:9742"
	req.Header.Set("Origin", "http://127.0.0.1:9742")
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Accept", "application/json, text/event-stream")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code == http.StatusForbidden {
		t.Fatalf("allowed origin was forbidden: %s", rr.Body.String())
	}
	if s.Hits() == 0 {
		t.Fatal("allowed /mcp did not reach the MCP handler")
	}
	if acao := rr.Header().Get("Access-Control-Allow-Origin"); acao != "" {
		t.Fatalf("tool POST set ACAO %q", acao)
	}
}

func TestServerInfoShape(t *testing.T) {
	s := testServer(t)
	info := s.Info()
	raw, err := json.Marshal(info)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"version", "port", "enabled", "default_location", "bind"} {
		if _, ok := m[k]; !ok {
			t.Errorf("missing %q in %s", k, raw)
		}
	}
	if info.Version != "1.2.0-test" || info.Port != DefaultPort || !info.Enabled || info.DefaultLocation != "New Haven, IN" || info.Bind != BindHost {
		t.Fatalf("info = %+v", info)
	}
	if _, ok := m["port"].(float64); !ok {
		t.Fatalf("port is not a number: %T", m["port"])
	}
	if m["bind"] != "127.0.0.1" {
		t.Fatalf("bind = %v", m["bind"])
	}
}

func TestPrepareDisabledAndPort(t *testing.T) {
	_, err := Prepare(ServeConfig{})
	if !errors.Is(err, ErrDisabled) {
		t.Fatalf("err = %v, want ErrDisabled", err)
	}
	opts, err := Prepare(ServeConfig{FlagEnable: true, Version: "1.2.0", DefaultLocation: "X"})
	if err != nil {
		t.Fatal(err)
	}
	if !opts.Enabled || opts.Port != DefaultPort || opts.Version != "1.2.0" || opts.DefaultLocation != "X" {
		t.Fatalf("opts = %+v", opts)
	}
	opts, err = Prepare(ServeConfig{ConfigEnabled: true, ConfigPort: 9750})
	if err != nil || opts.Port != 9750 {
		t.Fatalf("config port: %+v %v", opts, err)
	}
	opts, err = Prepare(ServeConfig{ConfigEnabled: true, ConfigPort: 9750, FlagPort: 9800, FlagPortSet: true})
	if err != nil || opts.Port != 9800 {
		t.Fatalf("flag port: %+v %v", opts, err)
	}
	_, err = Prepare(ServeConfig{FlagEnable: true, FlagPort: 0, FlagPortSet: true})
	if err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("port 0 flag: %v", err)
	}
	_, err = Prepare(ServeConfig{ConfigEnabled: true, ConfigPort: 70000})
	if err == nil {
		t.Fatal("expected invalid config port")
	}
	if _, err := ResolveListenPort(0, 0, false); err != nil {
		t.Fatal(err)
	}
}

func TestListenDisabledDoesNotBind(t *testing.T) {
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	err = ListenAndServe(context.Background(), Options{Enabled: false, Port: port, Version: "1.2.0"})
	if !errors.Is(err, ErrDisabled) {
		t.Fatalf("err = %v, want ErrDisabled", err)
	}
	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("disabled serve disturbed the existing listener: %v", err)
	}
	conn.Close()
}

func TestListenPortConflictFailsClosed(t *testing.T) {
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port
	var log strings.Builder
	err = ListenAndServe(context.Background(), Options{
		Enabled: true,
		Port:    port,
		Version: "1.2.0",
		Log:     &log,
	})
	if !errors.Is(err, ErrPortInUse) {
		t.Fatalf("err = %v, want ErrPortInUse", err)
	}
	if !strings.Contains(err.Error(), "refusing to pick another port") {
		t.Fatalf("error = %q", err.Error())
	}
	if !strings.Contains(err.Error(), fmt.Sprintf("127.0.0.1:%d", port)) {
		t.Fatalf("error = %q", err.Error())
	}
	if strings.Contains(log.String(), "listening") {
		t.Fatalf("conflict logged a ready line: %q", log.String())
	}
	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("conflict path closed the original listener: %v", err)
	}
	conn.Close()
}

func TestServeRejectsNonLoopbackListener(t *testing.T) {
	ln, err := net.Listen("tcp4", "0.0.0.0:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	s, err := New(Options{Enabled: true, Port: port, Version: "1.2.0"})
	if err != nil {
		t.Fatal(err)
	}
	err = s.Serve(context.Background(), ln)
	if err == nil || !strings.Contains(err.Error(), "127.0.0.1 only") {
		t.Fatalf("err = %v", err)
	}
	if err := ln.Close(); err == nil {
		t.Fatal("Serve left the wildcard listener open")
	}
}

func TestServerInfoOverStreamableHTTP(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	ln, err := net.Listen("tcp4", net.JoinHostPort(BindHost, "0"))
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	var log strings.Builder
	s, err := New(Options{
		Enabled:         true,
		Port:            port,
		Version:         "1.2.0",
		DefaultLocation: "New Haven, IN",
		Log:             &log,
	})
	if err != nil {
		t.Fatal(err)
	}
	errCh := make(chan error, 1)
	go func() { errCh <- s.Serve(ctx, ln) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-errCh:
		case <-time.After(5 * time.Second):
			t.Errorf("server did not exit")
		}
	})

	base := fmt.Sprintf("http://127.0.0.1:%d", port)
	deadline := time.Now().Add(3 * time.Second)
	var ready bool
	for time.Now().Before(deadline) {
		resp, err := http.Get(base + "/healthz")
		if err == nil {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK && string(body) == "1.2.0\n" {
				ready = true
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !ready {
		t.Fatalf("healthz not ready; log %q", log.String())
	}
	if !strings.Contains(log.String(), "127.0.0.1:"+fmt.Sprint(port)) {
		t.Fatalf("ready line = %q", log.String())
	}

	evil, err := http.NewRequest(http.MethodGet, base+"/healthz", nil)
	if err != nil {
		t.Fatal(err)
	}
	evil.Host = "evil.example"
	eresp, err := http.DefaultClient.Do(evil)
	if err != nil {
		t.Fatal(err)
	}
	ebody, _ := io.ReadAll(eresp.Body)
	eresp.Body.Close()
	if eresp.StatusCode != http.StatusForbidden {
		t.Fatalf("live forged host status = %d body %q", eresp.StatusCode, ebody)
	}

	originReq, err := http.NewRequest(http.MethodGet, base+"/mcp", nil)
	if err != nil {
		t.Fatal(err)
	}
	originReq.Host = "127.0.0.1:" + fmt.Sprint(port)
	originReq.Header.Set("Origin", "http://evil.example")
	oresp, err := http.DefaultClient.Do(originReq)
	if err != nil {
		t.Fatal(err)
	}
	obody, _ := io.ReadAll(oresp.Body)
	oresp.Body.Close()
	if oresp.StatusCode != http.StatusForbidden || strings.Contains(string(obody), "evil.example") {
		t.Fatalf("live forged origin status = %d body %q", oresp.StatusCode, obody)
	}

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "wx-mcp-test", Version: "0"}, nil)
	session, err := client.Connect(ctx, &sdkmcp.StreamableClientTransport{
		Endpoint: base + "/mcp",
	}, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer session.Close()

	res, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      "wx_server_info",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool error: %+v", res)
	}
	var text string
	for _, c := range res.Content {
		if tc, ok := c.(*sdkmcp.TextContent); ok {
			text = tc.Text
		}
	}
	if text == "" && res.StructuredContent != nil {
		raw, err := json.Marshal(res.StructuredContent)
		if err != nil {
			t.Fatal(err)
		}
		text = string(raw)
	}
	var info ServerInfo
	if err := json.Unmarshal([]byte(text), &info); err != nil {
		t.Fatalf("unmarshal %q: %v", text, err)
	}
	if info.Version != "1.2.0" || info.Port != port || !info.Enabled || info.DefaultLocation != "New Haven, IN" || info.Bind != "127.0.0.1" {
		t.Fatalf("info = %+v text %s", info, text)
	}
}
