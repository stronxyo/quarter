package steam

import "fmt"

// PlayURI starts a game through Steam.
func PlayURI(appID uint32) string { return fmt.Sprintf("steam://rungameid/%d", appID) }

// InstallURI opens Steam's install dialog for a game.
func InstallURI(appID uint32) string { return fmt.Sprintf("steam://install/%d", appID) }

// StoreURI opens a game's store page in the Steam client.
func StoreURI(appID uint32) string { return fmt.Sprintf("steam://store/%d", appID) }
