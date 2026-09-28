package cmd

import (
	"testing"
)

func TestAstroCommand_Metadata(t *testing.T) {
	cmd := astroCommand()
	if cmd.Name != "astro" {
		t.Errorf("expected name 'astro', got %q", cmd.Name)
	}

	expectedAliases := map[string]bool{"ephemeris": true, "sun": true, "moon": true}
	for _, alias := range cmd.Aliases {
		if !expectedAliases[alias] {
			t.Errorf("unexpected alias: %q", alias)
		}
		delete(expectedAliases, alias)
	}
	if len(expectedAliases) != 0 {
		t.Errorf("missing aliases: %v", expectedAliases)
	}

	// Verify required flags exist
	flagNames := make(map[string]bool)
	for _, f := range cmd.Flags {
		for _, name := range f.Names() {
			flagNames[name] = true
		}
	}

	for _, req := range []string{"location", "l", "date", "d", "json", "j"} {
		if !flagNames[req] {
			t.Errorf("missing expected flag %q", req)
		}
	}
}
