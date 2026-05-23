package monitor

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

//go:embed assets/logo.png
var logoBytes []byte

// getLogoPath returns the path to the extracted logo.png file in the user's config directory.
// It extracts the embedded logo to disk if it doesn't already exist.
func getLogoPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".config", "wx")
	logoPath := filepath.Join(dir, "logo.png")

	// Check if logo already exists on disk
	if _, err := os.Stat(logoPath); err == nil {
		return logoPath, nil
	}

	// Ensure the config directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	// Write the embedded bytes to disk
	if err := os.WriteFile(logoPath, logoBytes, 0644); err != nil {
		return "", err
	}

	return logoPath, nil
}

// notify displays a desktop notification with the app logo where supported.
func notify(title, message string) error {
	logoPath, _ := getLogoPath()

	switch runtime.GOOS {
	case "darwin":
		// macOS AppleScript display notification
		script := fmt.Sprintf("display notification %q with title %q", message, title)
		cmd := exec.Command("osascript", "-e", script)
		return cmd.Run()

	case "linux":
		// Linux notify-send with optional icon
		var args []string
		if logoPath != "" {
			args = append(args, "-i", logoPath)
		}
		args = append(args, title, message)
		cmd := exec.Command("notify-send", args...)
		return cmd.Run()

	case "windows":
		// Windows PowerShell notification with balloon tip
		script := fmt.Sprintf(
			`[void] [System.Reflection.Assembly]::LoadWithPartialName('System.Windows.Forms'); `+
				`$notification = New-Object System.Windows.Forms.NotifyIcon; `+
				`$notification.Icon = [System.Drawing.SystemIcons]::Information; `+
				`$notification.BalloonTipText = %q; `+
				`$notification.BalloonTipTitle = %q; `+
				`$notification.Visible = $True; `+
				`$notification.ShowBalloonTip(5000)`,
			message, title,
		)
		cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script)
		return cmd.Run()

	default:
		return nil
	}
}
