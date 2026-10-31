package cmd

import (
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
	for _, want := range []string{"--enable", "--port", "127.0.0.1", "9742"} {
		if !strings.Contains(text, want) {
			t.Errorf("help missing %q\n%s", want, text)
		}
	}
}
