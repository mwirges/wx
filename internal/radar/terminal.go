package radar

import "os"

// DetectTerminal returns the best image rendering protocol the current
// terminal supports. Falls back to TermHalfBlock for unknown terminals.
func DetectTerminal() TermCapability {
	// Kitty graphics protocol — Kitty terminal itself
	if os.Getenv("TERM") == "xterm-kitty" || os.Getenv("KITTY_PID") != "" {
		return TermKitty
	}

	switch os.Getenv("TERM_PROGRAM") {
	case "ghostty":
		// Ghostty supports the Kitty graphics protocol.
		return TermKitty
	case "iTerm.app", "WezTerm":
		// iTerm2 and WezTerm support the iTerm2 inline image protocol.
		return TermITerm2
	}

	return TermHalfBlock
}
