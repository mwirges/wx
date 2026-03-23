package radar

import "testing"

func TestDetectTerminal_ITerm2(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "iTerm.app")
	t.Setenv("TERM", "xterm-256color")
	if got := DetectTerminal(); got != TermITerm2 {
		t.Errorf("iTerm.app: got %d, want TermITerm2 (%d)", got, TermITerm2)
	}
}

func TestDetectTerminal_WezTerm(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "WezTerm")
	t.Setenv("TERM", "xterm-256color")
	if got := DetectTerminal(); got != TermITerm2 {
		t.Errorf("WezTerm: got %d, want TermITerm2 (%d)", got, TermITerm2)
	}
}

func TestDetectTerminal_Kitty(t *testing.T) {
	t.Setenv("TERM", "xterm-kitty")
	t.Setenv("TERM_PROGRAM", "")
	if got := DetectTerminal(); got != TermKitty {
		t.Errorf("xterm-kitty: got %d, want TermKitty (%d)", got, TermKitty)
	}
}

func TestDetectTerminal_KittyPID(t *testing.T) {
	t.Setenv("KITTY_PID", "12345")
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("TERM_PROGRAM", "")
	if got := DetectTerminal(); got != TermKitty {
		t.Errorf("KITTY_PID: got %d, want TermKitty (%d)", got, TermKitty)
	}
}

func TestDetectTerminal_Ghostty(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "ghostty")
	t.Setenv("TERM", "xterm-ghostty")
	if got := DetectTerminal(); got != TermKitty {
		t.Errorf("ghostty: got %d, want TermKitty (%d)", got, TermKitty)
	}
}

func TestDetectTerminal_Fallback(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "Apple_Terminal")
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("KITTY_PID", "")
	if got := DetectTerminal(); got != TermHalfBlock {
		t.Errorf("Apple_Terminal: got %d, want TermHalfBlock (%d)", got, TermHalfBlock)
	}
}
