package cmd

import (
	"testing"
)

func TestClimateCommand_Metadata(t *testing.T) {
	cmd := climateCommand()
	if cmd.Name != "climate" {
		t.Errorf("expected name 'climate', got %q", cmd.Name)
	}

	expectedAliases := map[string]bool{"normals": true, "records": true}
	for _, alias := range cmd.Aliases {
		if !expectedAliases[alias] {
			t.Errorf("unexpected alias: %q", alias)
		}
		delete(expectedAliases, alias)
	}
	if len(expectedAliases) != 0 {
		t.Errorf("missing aliases: %v", expectedAliases)
	}

	flagNames := make(map[string]bool)
	for _, f := range cmd.Flags {
		for _, name := range f.Names() {
			flagNames[name] = true
		}
	}

	for _, req := range []string{"location", "l", "date", "d", "units", "u", "json", "j"} {
		if !flagNames[req] {
			t.Errorf("missing expected flag %q", req)
		}
	}
}
