package library

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/stronxyo/quarter/internal/steam"
)

// Result is one full scan of the local Steam install.
type Result struct {
	SteamPath     string
	Games         []Game
	Uninstalled   []Game  // other apps Steam knows from this PC, unnamed until the store says what they are
	PlaytimeKnown bool    // false when the account or localconfig.vdf could not be read
	Warnings      []error `json:"-"` // files that failed to read; never cached
}

// Scan locates Steam (override may be empty) and builds the library from
// local files. It fails only when Steam itself can't be found.
func Scan(override string) (Result, error) {
	steamPath, err := steam.Locate(override)
	if err != nil {
		return Result{}, err
	}

	installed, warnings := steam.InstalledGames(steamPath)

	var stats map[uint32]steam.PlayStats
	var achievements map[uint32]steam.Achievements
	acct, err := steam.ActiveAccount(steamPath)
	if err == nil {
		// Achievements come from Steam's cached library pages; missing ones
		// just mean none are shown, so their errors aren't warnings.
		achievements, _ = steam.ReadLibraryPages(steamPath, acct)
		stats, err = steam.ReadPlayStats(steamPath, acct)
	}
	if err != nil {
		warnings = append(warnings, fmt.Errorf("playtime unavailable: %w", err))
	}

	return Result{
		SteamPath:     steamPath,
		Games:         FromSteam(installed, stats, achievements),
		Uninstalled:   uninstalled(installed, stats, achievements),
		PlaytimeKnown: stats != nil,
		Warnings:      warnings,
	}, nil
}

// uninstalled lists the apps Steam knows from this PC (they have play data in
// localconfig.vdf or a cached library page) that aren't installed and aren't
// Steam tools. Many are DLC or soundtracks; the store tells which are games.
func uninstalled(installed []steam.Manifest, stats map[uint32]steam.PlayStats, pages map[uint32]steam.Achievements) []Game {
	skip := make(map[uint32]bool, len(installed))
	for _, m := range installed {
		skip[m.AppID] = true
	}
	ids := make(map[uint32]bool)
	for id := range stats {
		ids[id] = true
	}
	for id := range pages {
		ids[id] = true
	}

	var out []Game
	for id := range ids {
		if skip[id] || !steam.IsGame(id) {
			continue
		}
		s := stats[id]
		out = append(out, Game{AppID: id, Playtime: s.Playtime, LastPlayed: s.LastPlayed, Achievements: pages[id]})
	}
	slices.SortFunc(out, func(a, b Game) int { return cmp.Compare(a.AppID, b.AppID) })
	return out
}
