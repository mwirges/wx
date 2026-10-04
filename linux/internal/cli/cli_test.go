package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mwirges/wx/linux/internal/fixture"
	"github.com/mwirges/wx/linux/internal/prefs"
)

func TestWeatherArgsMatchMacCLI(t *testing.T) {
	got := WeatherArgs("Fort Wayne, IN", "imperial", true)
	want := []string{"--json", "--forecast", "--alerts", "--hourly", "--hours", "24", "--location", "Fort Wayne, IN", "--units", "imperial"}
	if !same(got, want) {
		t.Fatalf("hourly args %v", got)
	}
	got = WeatherArgs("Home", "metric", false)
	want = []string{"--json", "--forecast", "--alerts", "--location", "Home", "--units", "metric"}
	if !same(got, want) {
		t.Fatalf("daily args %v", got)
	}
	for _, arg := range WeatherArgs("", "", false) {
		if arg == "--hourly" {
			t.Fatal("hourly flag without hourly")
		}
	}
}

func TestRadarAndExportArgsUseCLIWriters(t *testing.T) {
	got := RadarArgs("64101", "composite-reflectivity", 200, "", true, true, 8)
	want := []string{"radar", "--json", "--raw", "--loop", "--frames", "8", "--location", "64101", "--product", "composite-reflectivity", "--radius", "200"}
	if !same(got, want) {
		t.Fatalf("radar args %v", got)
	}
	got = ExportPNGArgs("/tmp/wx.png", "64101", "base-reflectivity", 150)
	want = []string{"radar", "--save", "/tmp/wx.png", "--location", "64101", "--product", "base-reflectivity", "--radius", "150"}
	if !same(got, want) {
		t.Fatalf("png args %v", got)
	}
	gif := ExportGIFArgs("/tmp/wx.gif", "64101", "echo-tops", 200, 8, 500)
	if !contains(gif, "--save-gif") || !contains(gif, "/tmp/wx.gif") {
		t.Fatalf("gif args %v", gif)
	}
	if contains(gif, "--json") {
		t.Fatalf("gif args include json %v", gif)
	}
	climate := ClimateArgs("Home", "imperial")
	if climate[0] != "climate" || climate[1] != "--json" {
		t.Fatalf("climate args %v", climate)
	}
}

func TestDecodeWeatherJSONFromStub(t *testing.T) {
	defer fixture.GuardHome(t)()
	dir := t.TempDir()
	binary := filepath.Join(dir, "wx")
	script := "#!/usr/bin/env python3\nimport json\nprint(json.dumps({\"conditions\": {\"temperature_f\": 72, \"location\": \"Stub\"}, \"alerts\": []}))\n"
	if err := os.WriteFile(binary, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	payload, err := New(binary).FetchWeather("Stub", "imperial", true)
	if err != nil {
		t.Fatal(err)
	}
	cond, _ := payload["conditions"].(map[string]any)
	if cond["temperature_f"] != 72.0 && cond["temperature_f"] != float64(72) {
		t.Fatalf("temp %#v", cond["temperature_f"])
	}
}

func TestCancelKillsRadarProcess(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "wx")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\nexec sleep 30\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	client := New(binary)
	errc := make(chan error, 1)
	go func() {
		_, err := client.FetchRadar("X", "composite-reflectivity", 200, "", 8)
		errc <- err
	}()
	time.Sleep(300 * time.Millisecond)
	client.CancelRadar()
	select {
	case err := <-errc:
		if !IsCancel(err) {
			t.Fatalf("err = %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("radar process was not cancelled")
	}
}

func TestMissingBinary(t *testing.T) {
	client := New("/tmp/wx-linux-missing-binary")
	if _, err := client.FetchWeather("", "", false); err == nil {
		t.Fatal("expected missing binary to fail")
	}
}

func TestExplicitBinaryIsNotReplaced(t *testing.T) {
	client := New("/tmp/wx-linux-missing-binary")
	if client.Binary() != "/tmp/wx-linux-missing-binary" {
		t.Fatalf("binary %q", client.Binary())
	}
}

func TestSuiteDoesNotTouchUserConfig(t *testing.T) {
	defer fixture.GuardHome(t)()
	t.Setenv("WX_CONFIG", filepath.Join(t.TempDir(), "wx-linux-unused.json"))
	_ = prefs.Load("")
}

func same(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func contains(args []string, needle string) bool {
	for _, arg := range args {
		if arg == needle {
			return true
		}
	}
	return false
}
