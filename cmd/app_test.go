package cmd

import (
	"strings"
	"testing"
)

func TestAppFlags_ExitCodeOnAlerts(t *testing.T) {
	app := NewApp()

	// Check top-level flag
	var foundTop bool
	for _, flag := range app.Flags {
		for _, name := range flag.Names() {
			if name == "exit-code-on-alerts" {
				foundTop = true
				break
			}
		}
	}
	if !foundTop {
		t.Errorf("expected --exit-code-on-alerts flag on root wx app")
	}

	// Check hourly subcommand flag
	var foundHourly bool
	for _, c := range app.Commands {
		if c.Name == "hourly" {
			for _, flag := range c.Flags {
				for _, name := range flag.Names() {
					if name == "exit-code-on-alerts" {
						foundHourly = true
						break
					}
				}
			}
		}
	}
	if !foundHourly {
		t.Errorf("expected --exit-code-on-alerts flag on hourly command")
	}
}

func TestAppHelp_ExitCodeOnAlerts(t *testing.T) {
	app := NewApp()
	out := captureStdout(t, func() {
		_ = app.Run([]string{"wx", "--help"})
	})
	if !strings.Contains(string(out), "--exit-code-on-alerts") {
		t.Errorf("expected --exit-code-on-alerts in wx --help output, got:\n%s", string(out))
	}
}
