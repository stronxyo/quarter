//go:build !windows

package steam

import (
	"os"
	"path/filepath"
)

// platformCandidates are the places Steam is found on Linux and macOS: the
// native installs, then Flatpak and Snap (the order protontricks uses).
func platformCandidates() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{
		filepath.Join(home, ".local", "share", "Steam"),
		filepath.Join(home, ".steam", "steam"),
		filepath.Join(home, ".steam", "root"),
		filepath.Join(home, ".var", "app", "com.valvesoftware.Steam", ".local", "share", "Steam"),
		filepath.Join(home, "snap", "steam", "common", ".local", "share", "Steam"),
		filepath.Join(home, ".snap", "data", "steam", "common", ".local", "share", "Steam"),
		filepath.Join(home, "Library", "Application Support", "Steam"),
	}
}

// realCase returns dir as is: only Windows reports paths in another casing.
func realCase(dir string) string { return dir }
