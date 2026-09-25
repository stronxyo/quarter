// Package steam reads the local Steam install: libraries, installed games and
// the active account's playtime. It never writes to Steam's folders.
package steam

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/stronxyo/quarter/internal/config"
)

// ErrNotFound means no Steam install was found.
var ErrNotFound = errors.New("steam install not found")

// Locate returns the Steam install directory. An explicit override wins, then
// the QUARTER_STEAM_PATH environment variable, then the platform's usual places.
func Locate(override string) (string, error) {
	if override != "" {
		return checkSteamDir(override)
	}
	if env := os.Getenv(config.SteamPathEnv); env != "" {
		return checkSteamDir(env)
	}
	for _, dir := range platformCandidates() {
		if isSteamDir(dir) {
			return realCase(filepath.Clean(dir)), nil
		}
	}
	return "", ErrNotFound
}

func checkSteamDir(dir string) (string, error) {
	if !isSteamDir(dir) {
		return "", ErrNotFound
	}
	return realCase(filepath.Clean(dir)), nil
}

func isSteamDir(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, "steamapps"))
	return err == nil && info.IsDir()
}
