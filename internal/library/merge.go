package library

import (
	"time"

	"github.com/stronxyo/quarter/internal/steam"
)

// FromSteam joins installed games with their play stats and achievements and
// sorts them by name. stats and achievements may be nil when unreadable.
func FromSteam(installed []steam.Manifest, stats map[uint32]steam.PlayStats, achievements map[uint32]steam.Achievements) []Game {
	games := make([]Game, 0, len(installed))
	for _, m := range installed {
		s := stats[m.AppID]
		games = append(games, Game{
			AppID:       m.AppID,
			Name:        m.Name,
			Installed:   true, // on disk in a library, or being put there
			Updating:    m.Updating(),
			Installing:  m.FirstInstall(),
			Paused:      m.Paused(),
			LibraryPath: m.LibraryPath,
			SizeOnDisk:  m.SizeOnDisk,
			Playtime:    s.Playtime,
			LastPlayed:  s.LastPlayed,

			Achievements: achievements[m.AppID],
		})
	}
	return View(games, FilterAll, SortName, false, time.Time{})
}
