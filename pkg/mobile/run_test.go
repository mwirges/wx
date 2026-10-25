package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
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
	}
	for _, tc := range cases {
		args, code, err := parseArgs(tc.in)
		if err != nil {
			t.Fatalf("parse %s: %s %v", tc.in, code, err)
		}
		got, err := buildArgv(args)
		if err != nil {
			t.Fatalf("build %s: %v", tc.in, err)
		}
		if strings.Join(got, "\x00") != strings.Join(tc.want, "\x00") {
			t.Fatalf("argv for %s\n got %q\nwant %q", tc.in, got, tc.want)
		}
		if got[len(got)-1] != "New Haven, IN" {
			t.Fatalf("location was split: %q", got)
		}
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
	t.Cleanup(func() {
		runTimeout = prevTimeout
		testDispatch = prevDispatch
	})

	runTimeout = 30 * time.Millisecond
	testDispatch = func(ctx context.Context, argv []string) (string, error) {
		<-ctx.Done()
		return "", ctx.Err()
	}

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
