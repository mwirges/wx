package packaging

import (
	"os"
	"strings"
	"testing"
)

func TestDesktopFileNamesTheApp(t *testing.T) {
	body, err := os.ReadFile("wx-desk.desktop")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, want := range []string{
		"Type=Application",
		"Name=wx",
		"Exec=wx-desk",
		"Icon=wx",
		"Terminal=false",
		"Categories=Science;Utility;",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %s", want)
		}
	}
}
