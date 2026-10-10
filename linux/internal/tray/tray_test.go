package tray

import (
	"testing"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/mwirges/wx/linux/internal/fixture"
	"github.com/mwirges/wx/linux/internal/present"
)

func TestTrayMenuHasMacEntries(t *testing.T) {
	menu := present.TrayMenu("Fort Wayne, IN", "72°", []map[string]any{{"name": "Home", "value": "Fort Wayne, IN"}}, "tactical", nil)
	labels := present.MenuLabels(menu)
	for _, want := range []string{
		"ACTIVE // FORT WAYNE, IN (72°)",
		"PINNED LOCATIONS // COMMAND GRID",
		"Home",
		"Menu Bar Format",
		present.FormatLabels["compact"],
		present.FormatLabels["standard"],
		present.FormatLabels["tactical"],
		"Open Desk Window",
		"Refresh Telemetry",
		"Quit wx",
	} {
		if !has(labels, want) {
			t.Fatalf("missing %q in %v", want, labels)
		}
	}
	numbered := NumberMenu(menu)
	flat := Flatten(numbered)
	var tactical *Node
	for _, node := range flat {
		if node.Props["label"] == present.FormatLabels["tactical"] {
			tactical = node
		}
	}
	if tactical == nil || tactical.Props["toggle-state"] != 1 {
		t.Fatalf("tactical toggle %#v", tactical)
	}
	sig := dbus.SignatureOf(Layout{})
	if sig.String() != "(ia{sv}av)" {
		t.Fatalf("signature %s", sig)
	}
	if numbered.ID != 0 {
		t.Fatalf("root id %d", numbered.ID)
	}
}

func TestTrayLabelAndIcon(t *testing.T) {
	tr := New(func() Snapshot {
		return Snapshot{Payload: fixture.Payload, Units: "imperial", Format: "standard", Location: "Fort Wayne, IN"}
	}, nil)
	if tr.StatusLabel() != " 🔴 72°" {
		t.Fatalf("label %q", tr.StatusLabel())
	}
	label, ok := tr.property(itemIface, "XAyatanaLabel")
	if !ok {
		t.Fatal("missing XAyatanaLabel")
	}
	if label.Value() != " 🔴 72°" {
		t.Fatalf("ayatana %v", label.Value())
	}
	guide, _ := tr.property(itemIface, "XAyatanaLabelGuide")
	if guide.Value() != "🔴 [WWWW] 000° ↘000mph" {
		t.Fatalf("guide %v", guide.Value())
	}
	icon, _ := tr.property(itemIface, "IconName")
	if icon.Value() != "weather-few-clouds" {
		t.Fatalf("icon %v", icon.Value())
	}
	if tr.revision != 1 {
		t.Fatalf("revision %d", tr.revision)
	}
	compact := New(func() Snapshot {
		return Snapshot{Payload: fixture.Payload, Units: "imperial", Format: "compact"}
	}, nil)
	icon, _ = compact.property(itemIface, "IconName")
	if icon.Value() != "" {
		t.Fatalf("compact icon %v", icon.Value())
	}
}

func TestMissingHostDoesNotBlock(t *testing.T) {
	tr := New(func() Snapshot { return Snapshot{Units: "imperial", Format: "standard"} }, nil)
	done := make(chan struct{})
	go func() {
		tr.Install()
		tr.Shutdown()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(4 * time.Second):
		t.Fatal("tray install blocked the caller")
	}
}

func has(labels []string, want string) bool {
	for _, label := range labels {
		if label == want {
			return true
		}
	}
	return false
}
