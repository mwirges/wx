package cmd

import (
	"testing"
)

func TestTropicsCommandFlags(t *testing.T) {
	cmd := tropicsCommand()

	if cmd.Name != "tropics" {
		t.Errorf("expected command name 'tropics', got %s", cmd.Name)
	}

	expectedAliases := []string{"nhc", "hurricane", "cyclone", "tropical"}
	for _, expected := range expectedAliases {
		found := false
		for _, a := range cmd.Aliases {
			if a == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected alias %s in %v", expected, cmd.Aliases)
		}
	}

	expectedFlags := []string{"location", "storm", "units", "json", "no-cache", "force-pretty"}
	for _, expected := range expectedFlags {
		found := false
		for _, f := range cmd.Flags {
			for _, name := range f.Names() {
				if name == expected {
					found = true
					break
				}
			}
		}
		if !found {
			t.Errorf("expected flag %s in tropicsCommand flags", expected)
		}
	}
}
