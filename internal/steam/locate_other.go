//go:build !windows

package steam

import (
	"os"
	"path/filepath"
)

func platformCandidates() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{
		filepath.Join(home, ".local", "share", "Steam"),
		filepath.Join(home, ".steam", "steam"),
		filepath.Join(home, "Library", "Application Support", "Steam"),
	}
}

// realCase returns dir as is: only Windows reports paths in another casing.
func realCase(dir string) string { return dir }
