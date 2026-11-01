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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/mwirges/wx/internal/config"
)

func TestMain(m *testing.M) {
	// Re-exec fixture: a copied binary named wx-cli blocks here so tests can
	// point a pidfile at a real "wx-cli mcp serve" process. It is not a server.
	if os.Getenv("WX_MCP_TEST_HELPER") == "1" {
		time.Sleep(30 * time.Second)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestMaySignalRefusesUnsafePIDs(t *testing.T) {
	good := ObservedProcess{
		PID:     4242,
		Alive:   true,
		ExePath: "/Applications/wx.app/Contents/MacOS/wx-cli",
		Command: "/Applications/wx.app/Contents/MacOS/wx-cli mcp serve --parent-pid 9",
	}
	cases := []struct {
		name string
		rec  PIDRecord
		obs  ObservedProcess
	}{
		{"pid 0", PIDRecord{PID: 0, Exe: good.ExePath, Args: []string{"mcp", "serve"}}, ObservedProcess{PID: 0, Alive: true, ExePath: good.ExePath, Command: good.Command}},
		{"pid -1", PIDRecord{PID: -1, Exe: good.ExePath}, ObservedProcess{PID: -1, Alive: true, Command: good.Command}},
		{"pid 1", PIDRecord{PID: 1, Exe: good.ExePath}, ObservedProcess{PID: 1, Alive: true, Command: good.Command}},
		{"mismatch", PIDRecord{PID: 7, Exe: good.ExePath, Args: []string{"mcp", "serve"}}, good},
		{"dead", PIDRecord{PID: good.PID, Exe: good.ExePath, Args: []string{"mcp", "serve"}}, ObservedProcess{PID: good.PID, Alive: false, ExePath: good.ExePath, Command: good.Command}},
		{"not wx", PIDRecord{PID: good.PID, Exe: "/bin/sleep", Args: []string{"mcp", "serve"}}, ObservedProcess{PID: good.PID, Alive: true, ExePath: "/bin/sleep", Command: "/bin/sleep mcp serve"}},
		{"wx hourly", PIDRecord{PID: good.PID, Exe: "/usr/local/bin/wx", Args: []string{"hourly"}}, ObservedProcess{PID: good.PID, Alive: true, ExePath: "/usr/local/bin/wx", Command: "/usr/local/bin/wx hourly"}},
		{"serve not adjacent", PIDRecord{PID: good.PID, Exe: good.ExePath}, ObservedProcess{PID: good.PID, Alive: true, ExePath: good.ExePath, Command: good.ExePath + " mcp --serve"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := MaySignal(tc.rec, tc.obs); err == nil {
				t.Fatal("MaySignal allowed an unsafe process")
			}
			if err := SignalVerified(tc.rec, tc.obs, syscall.SIGTERM); err == nil {
				t.Fatal("SignalVerified signaled an unsafe process")
			}
		})
	}
}

func TestMaySignalAllowsVerifiedServe(t *testing.T) {
	exe := "/tmp/wx.app/Contents/MacOS/wx-cli"
	cmd := exe + " mcp serve --config /tmp/config.json --parent-pid 42 --exit-on-stdin-close"
	rec := PIDRecord{PID: 88, Exe: exe, Argv0: exe, Args: []string{"mcp", "serve", "--parent-pid", "42"}}
	obs := ObservedProcess{PID: 88, Alive: true, ExePath: exe, Command: cmd}
	if err := MaySignal(rec, obs); err != nil {
		t.Fatal(err)
	}
	wx := "/usr/local/bin/wx"
	rec = PIDRecord{PID: 3, Exe: wx, Args: []string{"mcp", "serve"}}
	obs = ObservedProcess{PID: 3, Alive: true, ExePath: wx, Command: wx + " mcp serve --enable"}
	if err := MaySignal(rec, obs); err != nil {
		t.Fatal(err)
	}
	parent, ok := ParentPIDFromArgs(rec.Args)
	if !ok || parent != 0 {
		// rec.Args above has no parent. The first record does.
	}
	parent, ok = ParentPIDFromArgs([]string{"mcp", "serve", "--parent-pid", "42"})
	if !ok || parent != 42 {
		t.Fatalf("parent = %d %v", parent, ok)
	}
	if _, ok := ParentPIDFromArgs([]string{"--parent-pid", "0"}); ok {
		t.Fatal("accepted parent pid 0")
	}
	if _, ok := ParentPIDFromArgs([]string{"--parent-pid", "-1"}); ok {
		t.Fatal("accepted parent pid -1")
	}
}

func TestSignalVerifiedDoesNotKillForeignSleep(t *testing.T) {
	cmd := exec.Command("/bin/sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	t.Cleanup(func() {
		if pid > 1 {
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}
		_ = cmd.Wait()
	})
	obs, err := Observe(pid)
	if err != nil {
		t.Fatal(err)
	}
	if !obs.Alive || !strings.Contains(obs.Command, "sleep") {
		t.Fatalf("obs = %+v", obs)
	}
	rec := PIDRecord{PID: pid, Exe: "/bin/sleep", Args: []string{"mcp", "serve"}}
	if err := SignalVerified(rec, obs, syscall.SIGTERM); err == nil {
		t.Fatal("signaled sleep")
	}
	if !processAlive(pid) {
		t.Fatal("sleep is gone")
	}
}

func TestAcquireDoesNotKillForeignPIDFile(t *testing.T) {
	cmd := exec.Command("/bin/sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	t.Cleanup(func() {
		if pid > 1 {
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}
		_ = cmd.Wait()
	})
	dir := t.TempDir()
	_, pidPath := RuntimePaths(dir)
	if err := writePIDFile(pidPath, PIDRecord{PID: pid, Exe: "/bin/sleep", Args: []string{"mcp", "serve"}}); err != nil {
		t.Fatal(err)
	}
	lock, err := Acquire(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	if !processAlive(pid) {
		t.Fatal("Acquire killed the foreign pid")
	}
	got, err := ReadPIDFile(pidPath)
	if err != nil {
		t.Fatal(err)
	}
	if got.PID != os.Getpid() {
		t.Fatalf("pidfile pid = %d, want our pid %d", got.PID, os.Getpid())
	}
}

func TestAcquireRefusesLiveVerifiedServe(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "wx-cli")
	if err := os.Link(self, bin); err != nil {
		data, rerr := os.ReadFile(self)
		if rerr != nil {
			t.Fatal(err)
		}
		if werr := os.WriteFile(bin, data, 0o755); werr != nil {
			t.Fatal(werr)
		}
	}
	cmd := exec.Command(bin, "mcp", "serve")
	cmd.Env = append(os.Environ(), "WX_MCP_TEST_HELPER=1")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	t.Cleanup(func() {
		if pid > 1 {
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}
		_ = cmd.Wait()
	})
	deadline := time.Now().Add(2 * time.Second)
	var obs ObservedProcess
	for time.Now().Before(deadline) {
		obs, err = Observe(pid)
		if err == nil && obs.Alive && strings.Contains(obs.Command, "mcp") {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !obs.Alive || !strings.Contains(obs.Command, "mcp serve") {
		t.Fatalf("helper command = %+v err %v", obs, err)
	}
	rec := PIDRecord{PID: pid, Exe: bin, Argv0: bin, Args: []string{"mcp", "serve"}}
	if err := MaySignal(rec, obs); err != nil {
		t.Fatalf("MaySignal: %v command %q exe %q", err, obs.Command, obs.ExePath)
	}
	_, pidPath := RuntimePaths(dir)
	if err := writePIDFile(pidPath, rec); err != nil {
		t.Fatal(err)
	}
	lock, err := Acquire(dir)
	if lock != nil {
		lock.Release()
		t.Fatal("Acquire took the lock held by a verified serve record")
	}
	if !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("err = %v, want ErrAlreadyRunning", err)
	}
	if !processAlive(pid) {
		t.Fatal("Acquire killed the verified helper")
	}
	// SignalVerified may deliver SIGTERM. The Go helper can swallow that
	// signal, so a nil error is the assertion. Cleanup sends SIGKILL.
	if err := SignalVerified(rec, obs, syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
}

func TestSecondListenRefusesAndKeepsTheFirst(t *testing.T) {
	dir := t.TempDir()
	port := freePort(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		errCh <- ListenAndServe(ctx, Options{
			Enabled:    true,
			Port:       port,
			Version:    "1.2.0",
			RuntimeDir: dir,
		})
	}()
	waitHealth(t, port, "")
	err := ListenAndServe(context.Background(), Options{
		Enabled:    true,
		Port:       port,
		Version:    "1.2.0",
		RuntimeDir: dir,
	})
	if !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("second serve err = %v", err)
	}
	waitHealth(t, port, "")
	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("first serve did not exit")
	}
	ln, err := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("port still busy after stop: %v", err)
	}
	ln.Close()
	if _, err := os.Stat(filepath.Join(dir, pidFileName)); !os.IsNotExist(err) {
		t.Fatalf("pidfile left behind: %v", err)
	}
}

func TestStdinEOFClosesPort(t *testing.T) {
	r, w := io.Pipe()
	dir := t.TempDir()
	port := freePort(t)
	errCh := make(chan error, 1)
	go func() {
		errCh <- ListenAndServe(context.Background(), Options{
			Enabled:          true,
			Port:             port,
			Version:          "1.2.0",
			RuntimeDir:       dir,
			ExitOnStdinClose: true,
			Stdin:            r,
		})
	}()
	waitHealth(t, port, "")
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("stdin EOF did not stop serve")
	}
	ln, err := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("port still busy: %v", err)
	}
	ln.Close()
}

func TestParentExitClosesPort(t *testing.T) {
	parent := exec.Command("/bin/sleep", "30")
	if err := parent.Start(); err != nil {
		t.Fatal(err)
	}
	ppid := parent.Process.Pid
	t.Cleanup(func() {
		if ppid > 1 && processAlive(ppid) {
			_ = syscall.Kill(ppid, syscall.SIGKILL)
		}
		_ = parent.Wait()
	})
	dir := t.TempDir()
	port := freePort(t)
	errCh := make(chan error, 1)
	go func() {
		errCh <- ListenAndServe(context.Background(), Options{
			Enabled:    true,
			Port:       port,
			Version:    "1.2.0",
			RuntimeDir: dir,
			ParentPID:  ppid,
		})
	}()
	waitHealth(t, port, "")
	obs, err := Observe(ppid)
	if err != nil || !strings.Contains(obs.Command, "sleep") {
		t.Fatalf("parent obs = %+v err %v", obs, err)
	}
	if err := syscall.Kill(ppid, syscall.SIGKILL); err != nil {
		t.Fatal(err)
	}
	_ = parent.Wait()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("parent death did not stop serve")
	}
	ln, err := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("port still busy: %v", err)
	}
	ln.Close()
}

func TestParentPIDRefusesInitAndEveryone(t *testing.T) {
	// ParentPID 0 means "do not watch". 1 is init and -1 would be a
	// process-group signal if anyone passed it to kill. Both refuse
	// before a socket is opened.
	dir := t.TempDir()
	for _, pid := range []int{-1, 1} {
		port := freePort(t)
		err := ListenAndServe(context.Background(), Options{
			Enabled:    true,
			Port:       port,
			Version:    "1.2.0",
			RuntimeDir: dir,
			ParentPID:  pid,
		})
		if err == nil || !strings.Contains(err.Error(), "refusing") {
			t.Fatalf("parent %d err = %v", pid, err)
		}
		ln, lerr := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", port))
		if lerr != nil {
			t.Fatalf("parent %d bound anyway: %v", pid, lerr)
		}
		ln.Close()
	}
}

func TestDeadParentDoesNotBind(t *testing.T) {
	parent := exec.Command("/bin/sleep", "30")
	if err := parent.Start(); err != nil {
		t.Fatal(err)
	}
	pid := parent.Process.Pid
	_ = syscall.Kill(pid, syscall.SIGKILL)
	_ = parent.Wait()
	port := freePort(t)
	err := ListenAndServe(context.Background(), Options{
		Enabled:    true,
		Port:       port,
		Version:    "1.2.0",
		RuntimeDir: t.TempDir(),
		ParentPID:  pid,
	})
	if err == nil || !strings.Contains(err.Error(), "not running") {
		t.Fatalf("err = %v", err)
	}
	ln, lerr := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", port))
	if lerr != nil {
		t.Fatal(lerr)
	}
	ln.Close()
}

func TestRestartDelaySequence(t *testing.T) {
	want := []time.Duration{
		500 * time.Millisecond,
		time.Second,
		2 * time.Second,
		4 * time.Second,
		8 * time.Second,
	}
	for i, d := range want {
		got, ok := RestartDelay(i + 1)
		if !ok || got != d {
			t.Fatalf("attempt %d = %s %v, want %s", i+1, got, ok, d)
		}
	}
	if _, ok := RestartDelay(0); ok {
		t.Fatal("attempt 0 ran")
	}
	if _, ok := RestartDelay(6); ok {
		t.Fatal("attempt 6 ran")
	}
}

func TestTokenGuardOnHealthzAndMCP(t *testing.T) {
	const token = "unit-test-token-value"
	s, err := New(Options{
		Enabled:      true,
		Port:         DefaultPort,
		Version:      "1.2.0-test",
		RequireToken: true,
		Token:        token,
	})
	if err != nil {
		t.Fatal(err)
	}
	h := s.Handler()
	for _, path := range []string{"/healthz", "/mcp"} {
		before := s.Hits()
		rr := doReq(t, h, http.MethodGet, path, "127.0.0.1:9742", "", "")
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("%s missing bearer status = %d body %s", path, rr.Code, rr.Body.String())
		}
		assertUnauthorized(t, rr.Body.String(), token)
		if s.Hits() != before {
			t.Fatalf("%s unauthorized incremented hits", path)
		}
		req := httptestReq(t, http.MethodGet, path, "127.0.0.1:9742", "")
		req.Header.Set("Authorization", "Bearer wrong-token-value")
		rr = httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("%s wrong bearer status = %d", path, rr.Code)
		}
		assertUnauthorized(t, rr.Body.String(), token)
		if strings.Contains(rr.Body.String(), "wrong-token-value") {
			t.Fatal("response echoed the presented token")
		}
	}

	// A forged host is 403 even with the right bearer. Auth is not a bypass
	// of B1, and a 403 does not reveal whether the token matched.
	req := httptestReq(t, http.MethodGet, "/healthz", "evil.example", "")
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("evil host status = %d body %s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), token) {
		t.Fatal("403 body contains the token")
	}

	okReq := httptestReq(t, http.MethodGet, "/healthz", "127.0.0.1:9742", "")
	okReq.Header.Set("Authorization", "Bearer "+token)
	rr = httptest.NewRecorder()
	before := s.Hits()
	h.ServeHTTP(rr, okReq)
	if rr.Code != http.StatusOK || rr.Body.String() != "1.2.0-test\n" {
		t.Fatalf("healthz with bearer = %d %q", rr.Code, rr.Body.String())
	}
	if s.Hits() != before+1 {
		t.Fatalf("hits = %d", s.Hits())
	}
	localhost := httptestReq(t, http.MethodGet, "/healthz", "localhost:9742", "")
	localhost.Header.Set("Authorization", "bearer "+token)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, localhost)
	if rr.Code != http.StatusOK {
		t.Fatalf("lowercase bearer status = %d", rr.Code)
	}
}

func TestTokenFileModeAndSymlink(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.token")
	tok, err := GenerateToken()
	if err != nil {
		t.Fatal(err)
	}
	if len(tok) != 64 {
		t.Fatalf("token length %d", len(tok))
	}
	if err := WriteTokenFile(path, tok); err != nil {
		t.Fatal(err)
	}
	st, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %#o", st.Mode().Perm())
	}
	got, err := LoadTokenFile(path)
	if err != nil || got != tok {
		t.Fatalf("load %q err %v", got, err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadTokenFile(path); err == nil || !strings.Contains(err.Error(), "0600") {
		t.Fatalf("loose mode err = %v", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.token")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadTokenFile(link); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("symlink err = %v", err)
	}

	cfgPath := filepath.Join(dir, "config.json")
	require, loaded, err := LoadServeAuth(cfgPath, config.MCPSettings{RequireToken: true}, "")
	if err != nil || !require || loaded != tok {
		t.Fatalf("auth = %v %q %v", require, loaded, err)
	}
	require, loaded, err = LoadServeAuth(cfgPath, config.MCPSettings{}, "")
	if err != nil || require || loaded != "" {
		t.Fatalf("off = %v %q %v", require, loaded, err)
	}
	_, _, err = LoadServeAuth(cfgPath, config.MCPSettings{RequireToken: true}, link)
	if err == nil {
		t.Fatal("flag path followed a symlink")
	}
}

func TestRequireTokenFailClosedBeforeBind(t *testing.T) {
	port := freePort(t)
	err := ListenAndServe(context.Background(), Options{
		Enabled:      true,
		Port:         port,
		Version:      "1.2.0",
		RuntimeDir:   t.TempDir(),
		RequireToken: true,
		Token:        "",
	})
	if err == nil {
		t.Fatal("empty token listened")
	}
	ln, lerr := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", port))
	if lerr != nil {
		t.Fatal(lerr)
	}
	ln.Close()
}

func TestLiveTokenRequiredOnBothRoutes(t *testing.T) {
	const token = "unit-test-token-value"
	dir := t.TempDir()
	port := freePort(t)
	var log strings.Builder
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		errCh <- ListenAndServe(ctx, Options{
			Enabled:      true,
			Port:         port,
			Version:      "1.2.0",
			RuntimeDir:   dir,
			RequireToken: true,
			Token:        token,
			Log:          &log,
		})
	}()
	waitHealth(t, port, token)
	if strings.Contains(log.String(), token) {
		t.Fatal("ready log contains the token")
	}
	for _, path := range []string{"/healthz", "/mcp"} {
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d%s", port, path), nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Host = fmt.Sprintf("127.0.0.1:%d", port)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("%s status = %d body %s", path, resp.StatusCode, body)
		}
		assertUnauthorized(t, string(body), token)
	}
	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not exit")
	}
}

func TestPIDFileRejectsTinyPID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, pidFileName)
	if err := os.WriteFile(path, []byte(`{"pid":0,"exe":"/usr/local/bin/wx","args":["mcp","serve"]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadPIDFile(path); err == nil {
		t.Fatal("accepted pid 0")
	}
	if err := writePIDFile(path, PIDRecord{PID: -1, Exe: "/usr/local/bin/wx"}); err == nil {
		t.Fatal("wrote pid -1")
	}
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	return port
}

func waitHealth(t *testing.T, port int, token string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	var last string
	for time.Now().Before(deadline) {
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/healthz", port), nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Host = fmt.Sprintf("127.0.0.1:%d", port)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
			last = fmt.Sprintf("%d %s", resp.StatusCode, body)
		} else {
			last = err.Error()
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("healthz not ready: %s", last)
}

func assertUnauthorized(t *testing.T, body, token string) {
	t.Helper()
	if strings.Contains(body, token) {
		t.Fatal("unauthorized body contains the token")
	}
	var payload struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatalf("body %q: %v", body, err)
	}
	if payload.Code != "unauthorized" || payload.Message == "" {
		t.Fatalf("payload = %+v", payload)
	}
}

func httptestReq(t *testing.T, method, path, host, body string) *http.Request {
	t.Helper()
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, "http://127.0.0.1:9742"+path, rdr)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = host
	return req
}
