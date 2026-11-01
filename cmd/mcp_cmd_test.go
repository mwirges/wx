package cmd

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestMCPHelp(t *testing.T) {
	app := NewApp()
	var found bool
	for _, c := range app.Commands {
		if c.Name == "mcp" {
			found = true
			if len(c.Subcommands) != 1 || c.Subcommands[0].Name != "serve" {
				t.Fatalf("mcp subcommands = %v", c.Subcommands)
			}
		}
	}
	if !found {
		t.Fatal("wx mcp command is not registered")
	}

	out := captureStdout(t, func() {
		_ = app.Run([]string{"wx", "mcp", "serve", "--help"})
	})
	text := string(out)
	for _, want := range []string{"--enable", "--port", "--config", "--token-file", "--parent-pid", "--exit-on-stdin-close", "127.0.0.1", "9742", "0600"} {
		if !strings.Contains(text, want) {
			t.Errorf("help missing %q\n%s", want, text)
		}
	}
}

func TestMCPParentPIDZeroDoesNotListen(t *testing.T) {
	app := NewApp()
	dir := t.TempDir()
	err := app.Run([]string{
		"wx", "mcp", "serve", "--enable",
		"--parent-pid", "0",
		"--config", filepath.Join(dir, "config.json"),
	})
	if err == nil || !strings.Contains(err.Error(), "refusing") {
		t.Fatalf("err = %v", err)
	}
	err = app.Run([]string{
		"wx", "mcp", "serve", "--enable",
		"--parent-pid", "-1",
		"--config", filepath.Join(dir, "config.json"),
	})
	if err == nil || !strings.Contains(err.Error(), "refusing") {
		t.Fatalf("pid -1 err = %v", err)
	}
}
