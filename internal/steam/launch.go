package steam

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// PlayURI starts a game through Steam.
func PlayURI(appID uint32) string { return fmt.Sprintf("steam://rungameid/%d", appID) }

// InstallURI opens Steam's install dialog for a game.
func InstallURI(appID uint32) string { return fmt.Sprintf("steam://install/%d", appID) }

// UninstallURI asks Steam to uninstall a game (community documented; Quarter
// asks the user to press twice first, in case Steam doesn't ask).
func UninstallURI(appID uint32) string { return fmt.Sprintf("steam://uninstall/%d", appID) }

// StoreURI opens a game's store page in the Steam client.
func StoreURI(appID uint32) string { return fmt.Sprintf("steam://store/%d", appID) }

// LibraryURI opens a game's page in the Steam library. Valve documents
// steam://nav/games/details/<id> (and steam://open/games/details/<id> worked
// as well on Pablo's PC, 2026-10-01): the documented one, for any OS. For a
// game not in the library Steam does nothing, and shows the game it showed
// last: open LibraryHomeURI first.
func LibraryURI(appID uint32) string { return fmt.Sprintf("steam://nav/games/details/%d", appID) }

// LibraryHomeURI opens the Steam library's home (steam://nav/games; worked
// on Pablo's PC, as steam://open/games did).
const LibraryHomeURI = "steam://nav/games"

// LibraryCacheDir is where Steam keeps a game's library artwork. Steam has
// one for the games in the library; a game without one (most demos,
// playtests and games no longer sold) can't be shown there. A sign, not a
// record: an old folder can outlive the game in the library.
func LibraryCacheDir(steamPath string, appID uint32) string {
	return filepath.Join(steamPath, "appcache", "librarycache", strconv.FormatUint(uint64(appID), 10))
}

// InLibraryCache reports whether Steam keeps library artwork for the game:
// the folder exists. It reads the disk.
func InLibraryCache(steamPath string, appID uint32) bool {
	info, err := os.Stat(LibraryCacheDir(steamPath, appID))
	return err == nil && info.IsDir()
}
