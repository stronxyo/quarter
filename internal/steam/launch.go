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
