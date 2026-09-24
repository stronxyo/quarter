package config

import (
	"os"
	"path/filepath"
)

// DataDir returns Quarter's folder for cache, config and logs, creating it if
// needed: %APPDATA%\quarter on Windows, ~/.config/quarter on Linux and
// ~/Library/Application Support/quarter on macOS.
func DataDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, AppName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}
