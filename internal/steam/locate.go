// Package steam reads the local Steam install: libraries, installed games and
// the active account's playtime. It never writes to Steam's folders.
package steam

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/stronxyo/quarter/internal/config"
)

// ErrNotFound means no Steam install was found.
var ErrNotFound = errors.New("steam install not found")

// NotFoundError says where Quarter looked for Steam. Given is the folder the
// user named (--steam-path or QUARTER_STEAM_PATH), "" when Quarter searched
// the usual places, which are then in Looked.
type NotFoundError struct {
	Given  string
	Looked []string
}

func (e *NotFoundError) Error() string {
	if e.Given != "" {
		return e.Given + " is not a Steam folder: it has no steamapps folder"
	}
	return "steam install not found in " + strings.Join(e.Looked, ", ")
}

func (e *NotFoundError) Is(target error) bool { return target == ErrNotFound }

// Locate returns the Steam install directory. An explicit override wins, then
// the QUARTER_STEAM_PATH environment variable, then the platform's usual places.
func Locate(override string) (string, error) {
	if override != "" {
		return checkSteamDir(override)
	}
	if env := os.Getenv(config.SteamPathEnv); env != "" {
		return checkSteamDir(env)
	}
	var looked []string
	for _, dir := range platformCandidates() {
		if isSteamDir(dir) {
			return realCase(filepath.Clean(dir)), nil
		}
		dir = filepath.Clean(dir)
		if !slices.ContainsFunc(looked, func(d string) bool { return strings.EqualFold(d, dir) }) {
			looked = append(looked, dir)
		}
	}
	return "", &NotFoundError{Looked: looked}
}

func checkSteamDir(dir string) (string, error) {
	if !isSteamDir(dir) {
		return "", &NotFoundError{Given: filepath.Clean(dir)}
	}
	return realCase(filepath.Clean(dir)), nil
}

func isSteamDir(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, "steamapps"))
	return err == nil && info.IsDir()
}
