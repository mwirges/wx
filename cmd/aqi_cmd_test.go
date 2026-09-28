package cmd

import (
	"testing"
)

func TestAQICommand_Metadata(t *testing.T) {
	cmd := aqiCommand()
	if cmd.Name != "aqi" {
		t.Errorf("expected name 'aqi', got %q", cmd.Name)
	}

	expectedAliases := map[string]bool{"air": true, "airquality": true, "smoke": true}
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

	for _, req := range []string{"location", "l", "no-cache", "json", "j"} {
		if !flagNames[req] {
			t.Errorf("missing expected flag %q", req)
		}
	}
}
