package library

import (
	"sort"
	"strings"

	"github.com/stronxyo/quarter/internal/steam"
)

// FromSteam joins installed games with their play stats and sorts them by
// name. stats may be nil when play data could not be read.
func FromSteam(installed []steam.Manifest, stats map[uint32]steam.PlayStats) []Game {
	games := make([]Game, 0, len(installed))
	for _, m := range installed {
		s := stats[m.AppID]
		games = append(games, Game{
			AppID:       m.AppID,
			Name:        m.Name,
			Installed:   m.Installed(),
			LibraryPath: m.LibraryPath,
			SizeOnDisk:  m.SizeOnDisk,
			Playtime:    s.Playtime,
			LastPlayed:  s.LastPlayed,
		})
	}
	sort.SliceStable(games, func(i, j int) bool {
		return strings.ToLower(games[i].Name) < strings.ToLower(games[j].Name)
	})
	return games
}
