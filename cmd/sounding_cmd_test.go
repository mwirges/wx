package cmd

import (
	"testing"
)

func TestSoundingCommand_Metadata(t *testing.T) {
	cmd := soundingCommand()
	if cmd.Name != "sounding" {
		t.Errorf("expected command name 'sounding', got %q", cmd.Name)
	}

	expectedAliases := map[string]bool{"cape": true, "instability": true}
	for _, alias := range cmd.Aliases {
		delete(expectedAliases, alias)
	}
	if len(expectedAliases) > 0 {
		t.Errorf("missing expected aliases: %v", expectedAliases)
	}

	var hasStation, hasJSON, hasUnits, hasLocation bool
	for _, flag := range cmd.Flags {
		for _, name := range flag.Names() {
			switch name {
			case "station", "s":
				hasStation = true
			case "json", "j":
				hasJSON = true
			case "units", "u":
				hasUnits = true
			case "location", "l":
				hasLocation = true
			}
		}
	}

	if !hasStation {
		t.Error("expected --station flag on sounding command")
	}
	if !hasJSON {
		t.Error("expected --json flag on sounding command")
	}
	if !hasUnits {
		t.Error("expected --units flag on sounding command")
	}
	if !hasLocation {
		t.Error("expected --location flag on sounding command")
	}
}
