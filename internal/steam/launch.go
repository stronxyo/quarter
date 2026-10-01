package steam

import "fmt"

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
// as well on Pablo's PC, 2026-10-01): the documented one, for any OS.
func LibraryURI(appID uint32) string { return fmt.Sprintf("steam://nav/games/details/%d", appID) }
