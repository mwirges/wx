package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mwirges/wx/cmd"
)

func TestDefaultTimeout(t *testing.T) {
	if runTimeout != 15*time.Second {
		t.Fatalf("default timeout = %s, want 15s", runTimeout)
	}
}

func TestBuildArgv_NewHavenIsOneArgument(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{
			`["forecast","New Haven, IN"]`,
			[]string{"wx", "--forecast", "--json", "New Haven, IN"},
		},
		{
			`["alerts","New Haven, IN"]`,
			[]string{"wx", "--alerts", "--json", "New Haven, IN"},
		},
		{
			`["hourly","New Haven, IN"]`,
			[]string{"wx", "hourly", "--json", "New Haven, IN"},
		},
		{
			`["nowcast","New Haven, IN"]`,
			[]string{"wx", "nowcast", "--json", "New Haven, IN"},
		},
		{
			`["radar","--no-cache","New Haven, IN"]`,
			[]string{"wx", "radar", "--json", "--no-cache", "New Haven, IN"},
		},
		{
			`["spc","New Haven, IN"]`,
			[]string{"wx", "spc", "--json", "New Haven, IN"},
		},
		{
			`["climate","New Haven, IN"]`,
			[]string{"wx", "climate", "--json", "New Haven, IN"},
		},
		{
			`["tropics","New Haven, IN"]`,
			[]string{"wx", "tropics", "--json", "New Haven, IN"},
		},
		{
			`["aqi","New Haven, IN"]`,
			[]string{"wx", "aqi", "--json", "New Haven, IN"},
		},
		{
			`["astro","--date","2026-06-21","New Haven, IN"]`,
			[]string{"wx", "astro", "--json", "--date", "2026-06-21", "New Haven, IN"},
		},
		{
			`["forecast","--json","New Haven, IN"]`,
			[]string{"wx", "--forecast", "--json", "New Haven, IN"},
		},
		{
			`["forecast","New Haven, IN","--json"]`,
			[]string{"wx", "--forecast", "--json", "New Haven, IN"},
		},
		{
			`["spc","New Haven, IN","--json"]`,
			[]string{"wx", "spc", "--json", "New Haven, IN"},
		},
		{
			`["aqi","-j","New Haven, IN"]`,
			[]string{"wx", "aqi", "--json", "New Haven, IN"},
		},
		{
			`["radar","--json","--no-cache","New Haven, IN"]`,
			[]string{"wx", "radar", "--json", "--no-cache", "New Haven, IN"},
		},
	}
	for _, tc := range cases {
		args, code, err := parseArgs(tc.in)
		if err != nil {
			t.Fatalf("parse %s: %s %v", tc.in, code, err)
		}
		got, code, err := buildArgv(args)
		if err != nil {
			t.Fatalf("build %s: %s %v", tc.in, code, err)
		}
		if strings.Join(got, "\x00") != strings.Join(tc.want, "\x00") {
			t.Fatalf("argv for %s\n got %q\nwant %q", tc.in, got, tc.want)
		}
		if got[len(got)-1] != "New Haven, IN" {
			t.Fatalf("location was split: %q", got)
		}
	}
}

func TestRunJSON_PositionalJSONProducesJSON(t *testing.T) {
	prev := testDispatch
	t.Cleanup(func() { testDispatch = prev })

	var mu sync.Mutex
	var seen [][]string
	testDispatch = func(ctx context.Context, argv []string) (string, error) {
		mu.Lock()
		seen = append(seen, append([]string{}, argv...))
		mu.Unlock()
		return `{"location":"New Haven, IN","ok":true}` + "\n", nil
	}

	inputs := []string{
		`["spc","New Haven, IN","--json"]`,
		`["aqi","New Haven, IN"]`,
	}
	for _, in := range inputs {
		out := RunJSON(in)
		if !json.Valid([]byte(out)) {
			t.Fatalf("%s: not JSON: %s", in, out)
		}
		var body map[string]any
		if err := json.Unmarshal([]byte(out), &body); err != nil {
			t.Fatalf("%s: %v", in, err)
		}
		if _, bad := body["error"]; bad {
			t.Fatalf("%s: error JSON: %s", in, out)
		}
		if body["location"] != "New Haven, IN" {
			t.Fatalf("%s: location = %#v", in, body["location"])
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if len(seen) != 2 {
		t.Fatalf("dispatched %d times", len(seen))
	}
	for _, argv := range seen {
		if len(argv) < 4 || argv[2] != "--json" || argv[3] != "New Haven, IN" {
			t.Fatalf("argv = %q", argv)
		}
		for i, arg := range argv[3:] {
			if arg == "--json" || arg == "-j" {
				t.Fatalf("--json followed the place name at %d: %q", i, argv)
			}
		}
	}
}

func TestRunJSON_RejectsNonJSONFlags(t *testing.T) {
	prev := testDispatch
	t.Cleanup(func() { testDispatch = prev })
	called := false
	testDispatch = func(ctx context.Context, argv []string) (string, error) {
		called = true
		return "", nil
	}
	cases := []string{
		`["forecast","--short","New Haven, IN"]`,
		`["hourly","--template","{{.Location}}"]`,
		`["hourly","-T","{{.Location}}"]`,
		`["radar","--save","/tmp/wx.png","New Haven, IN"]`,
		`["radar","--save=/tmp/wx.png","New Haven, IN"]`,
		`["radar","--save-gif","/tmp/wx.gif"]`,
		`["radar","--interactive","New Haven, IN"]`,
		`["radar","--no-inline","New Haven, IN"]`,
		`["tropics","--force-pretty","New Haven, IN"]`,
		`["alerts","--exit-code-on-alerts","New Haven, IN"]`,
		`["forecast","-s","New Haven, IN"]`,
	}
	for _, in := range cases {
		assertErrorCode(t, RunJSON(in), "invalid_args")
	}
	if called {
		t.Fatal("rejected flag was dispatched")
	}
}

func TestRunJSON_RadarLoopReturnsFrames(t *testing.T) {
	prev := testDispatch
	t.Cleanup(func() { testDispatch = prev })
	cases := []struct {
		in   string
		want []string
	}{
		{
			`["radar","--loop","--frames","6","--raw","--location","New Haven, IN"]`,
			[]string{"wx", "radar", "--json", "--loop", "--frames", "6", "--raw", "--location", "New Haven, IN"},
		},
		{
			`["radar","-L","New Haven, IN"]`,
			[]string{"wx", "radar", "--json", "-L", "New Haven, IN"},
		},
	}
	for _, tc := range cases {
		var got []string
		testDispatch = func(ctx context.Context, argv []string) (string, error) {
			got = append([]string{}, argv...)
			return `{"ok":true}`, nil
		}
		out := RunJSON(tc.in)
		if strings.Contains(out, `"error"`) {
			t.Fatalf("RunJSON(%s) = %s", tc.in, out)
		}
		if strings.Join(got, " ") != strings.Join(tc.want, " ") {
			t.Fatalf("argv = %q, want %q", got, tc.want)
		}
	}
}

func TestRunJSON_BlockedCaptureTimesOutWithoutLeak(t *testing.T) {
	prevTimeout := runTimeout
	prevBody := cmd.TestCaptureBody
	runTimeout = 100 * time.Millisecond

	started := make(chan struct{})
	release := make(chan struct{})
	var stopOnce sync.Once
	stop := func() { stopOnce.Do(func() { close(release) }) }
	var seq atomic.Int32
	cmd.TestCaptureBody = func(ctx context.Context) error {
		if seq.Add(1) == 1 {
			close(started)
			<-ctx.Done()
			fmt.Fprintln(os.Stdout, "SECRET-LATE")
			<-release
			return nil
		}
		fmt.Fprintln(os.Stdout, "CALL-TWO")
		return nil
	}
	t.Cleanup(func() {
		stop()
		cmd.TestCaptureBody = prevBody
		runTimeout = prevTimeout
	})

	type call struct {
		out     string
		elapsed time.Duration
	}
	firstCh := make(chan call, 1)
	go func() {
		start := time.Now()
		out := RunJSON(`["forecast","New Haven, IN"]`)
		firstCh <- call{out: out, elapsed: time.Since(start)}
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("first capture did not start")
	}

	secondStart := time.Now()
	second := RunJSON(`["aqi","New Haven, IN"]`)
	secondElapsed := time.Since(secondStart)

	var first call
	select {
	case first = <-firstCh:
	case <-time.After(2 * time.Second):
		t.Fatal("first call did not return while the capture was still held")
	}

	// Restore stdout before reporting a failure. The first command is still
	// holding the process stdout until stop runs.
	stop()
	runTimeout = 2 * time.Second
	third := RunJSON(`["hourly","New Haven, IN"]`)

	if first.elapsed > time.Second {
		t.Fatalf("blocked call took %s, want timeout within 1s", first.elapsed)
	}
	assertErrorCode(t, first.out, "timeout")
	if strings.Contains(first.out, "SECRET") {
		t.Fatalf("timeout result included late stdout: %s", first.out)
	}
	if secondElapsed > time.Second {
		t.Fatalf("second call waited %s on the blocked capture", secondElapsed)
	}
	assertErrorCode(t, second, "timeout")
	if strings.Contains(second, "SECRET") || strings.Contains(second, "CALL-TWO") {
		t.Fatalf("second call leaked output: %s", second)
	}
	if strings.Contains(third, "SECRET") {
		t.Fatalf("late stdout leaked into the next capture: %s", third)
	}
	if !strings.Contains(third, "CALL-TWO") {
		t.Fatalf("third capture = %q", third)
	}
}

func TestRunJSON_Errors(t *testing.T) {
	cases := []struct {
		in   string
		code string
	}{
		{``, "invalid_json"},
		{`not json`, "invalid_json"},
		{`{"forecast":"New Haven, IN"}`, "invalid_json"},
		{`[]`, "invalid_args"},
		{`["forecast", 1]`, "invalid_args"},
		{`["chase","New Haven, IN"]`, "unknown_command"},
		{`["monitor"]`, "unknown_command"},
		{`["config"]`, "unknown_command"},
		{`["forecast","--not-a-real-flag"]`, "command_error"},
	}
	for _, tc := range cases {
		assertErrorCode(t, RunJSON(tc.in), tc.code)
	}
}

func TestRunJSON_Timeout(t *testing.T) {
	prevTimeout := runTimeout
	prevDispatch := testDispatch
	runTimeout = 30 * time.Millisecond
	done := make(chan struct{})
	testDispatch = func(ctx context.Context, argv []string) (string, error) {
		defer close(done)
		<-ctx.Done()
		return "", ctx.Err()
	}
	// The timeout returns before this goroutine finishes. Wait for it before
	// restoring testDispatch so the race detector sees the happens-before.
	t.Cleanup(func() {
		<-done
		runTimeout = prevTimeout
		testDispatch = prevDispatch
	})

	start := time.Now()
	out := RunJSON(`["forecast","New Haven, IN"]`)
	if time.Since(start) > 2*time.Second {
		t.Fatalf("timeout took too long: %s", time.Since(start))
	}
	assertErrorCode(t, out, "timeout")
}

func TestRunJSON_Panic(t *testing.T) {
	prev := testDispatch
	t.Cleanup(func() { testDispatch = prev })
	testDispatch = func(ctx context.Context, argv []string) (string, error) {
		panic("boom")
	}
	out := RunJSON(`["aqi","New Haven, IN"]`)
	assertErrorCode(t, out, "panic")
	var body struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(out), &body); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body.Error.Message, "boom") {
		t.Fatalf("message = %q", body.Error.Message)
	}
}

func TestRunJSON_CommandErrorDoesNotDispatchUnknown(t *testing.T) {
	prev := testDispatch
	t.Cleanup(func() { testDispatch = prev })
	called := false
	testDispatch = func(ctx context.Context, argv []string) (string, error) {
		called = true
		return "", nil
	}
	assertErrorCode(t, RunJSON(`["outlook"]`), "unknown_command")
	if called {
		t.Fatal("unknown command was dispatched")
	}
}

func TestRunJSON_UnwritableHomeCacheIsCommandError(t *testing.T) {
	home := t.TempDir()
	dot := filepath.Join(home, ".cache")
	if err := os.Mkdir(dot, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dot, 0o755) })
	t.Setenv("HOME", home)
	t.Setenv("WX_CACHE_DIR", "")

	out := RunJSON(`["forecast","--location","New Haven, IN"]`)
	assertErrorCode(t, out, "command_error")
	if !strings.Contains(out, "cache init") {
		t.Fatalf("bridge hid the cache failure: %s", out)
	}
}

func assertErrorCode(t *testing.T, out, code string) {
	t.Helper()
	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(out), &body); err != nil {
		t.Fatalf("error JSON %s: %v", out, err)
	}
	if body.Error.Code != code {
		t.Fatalf("code = %q, want %q (%s)", body.Error.Code, code, out)
	}
	if body.Error.Message == "" {
		t.Fatalf("empty message for %s: %s", code, out)
	}
}
