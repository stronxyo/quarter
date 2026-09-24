package steam

import "github.com/stronxyo/quarter/internal/config"

// IsGame reports whether an app is a game rather than a Steam tool such as
// Proton or the common redistributables.
func IsGame(appID uint32) bool {
	_, tool := config.NonGameAppIDs[appID]
	return !tool
}
