package library

import (
	"fmt"

	"github.com/stronxyo/quarter/internal/steam"
)

// Result is one full scan of the local Steam install.
type Result struct {
	SteamPath     string
	Games         []Game
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
		achievements, _ = steam.ReadAchievements(steamPath, acct)
		stats, err = steam.ReadPlayStats(steamPath, acct)
	}
	if err != nil {
		warnings = append(warnings, fmt.Errorf("playtime unavailable: %w", err))
	}

	return Result{
		SteamPath:     steamPath,
		Games:         FromSteam(installed, stats, achievements),
		PlaytimeKnown: stats != nil,
		Warnings:      warnings,
	}, nil
}
